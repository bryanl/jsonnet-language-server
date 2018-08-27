package token

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NodeCacheMissErr is an error for a cache miss.
type NodeCacheMissErr struct {
	key string
}

func (e *NodeCacheMissErr) Error() string {
	return fmt.Sprintf("%q did not exist in cache", e.key)
}

// NodeCacheDependency is a depedency of a cached item.
type NodeCacheDependency struct {
	Name      string
	UpdatedAt time.Time
}

// NodeEntry is an entry in the NodeCache.
type NodeEntry struct {
	Node         ast.Node
	Dependencies []NodeCacheDependency

	libPaths []string
	filename string
}

// NewNodeEntry creates an instance of NodeEntry.
func NewNodeEntry(deps []NodeCacheDependency, libPaths []string, filename string) *NodeEntry {
	return &NodeEntry{
		Dependencies: deps,
		libPaths:     libPaths,
		filename:     filename,
	}
}

// NodeCache is a cache for nodes.
type NodeCache struct {
	store       map[string]NodeEntry
	nodeBuilder NodeBuilder

	mu sync.Mutex
}

// NewNodeCache creates an instance of NodeCache.
func NewNodeCache() *NodeCache {
	c := &NodeCache{
		store:       make(map[string]NodeEntry),
		nodeBuilder: &nodeBuilder{},
	}

	return c
}

// Keys returns a list of keys in the cache.
func (c *NodeCache) Keys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	var keys []string
	for k := range c.store {
		keys = append(keys, k)
	}

	return keys
}

// Get gets a key from the cache.
func (c *NodeCache) Get(key string) (*NodeEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.store[key]
	if !ok {
		return nil, &NodeCacheMissErr{key: key}
	}

	return &e, nil
}

// Set sets a key in the cache.
func (c *NodeCache) Set(key string, e *NodeEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	existing, ok := c.store[key]
	if !ok {
		logrus.WithField("key", key).Info("setting new cache entry")
		return c.set(key, e)
	}

	isUpdate := false

	for _, dep := range e.Dependencies {
		for _, existingDep := range existing.Dependencies {
			if existingDep.Name == dep.Name &&
				dep.UpdatedAt.After(existingDep.UpdatedAt) {
				isUpdate = true
			}
		}
	}

	if isUpdate {
		logrus.WithField("key", key).Info("updating existing cache entry")
		return c.set(key, e)
	}

	logrus.WithField("key", key).Info("cache entry is up to date")
	return nil
}

func (c *NodeCache) set(key string, e *NodeEntry) error {
	now := time.Now()
	defer func() {
		logrus.WithField("elapsed", time.Since(now)).Info("node evaluate time")
	}()

	node, err := c.nodeBuilder.Build(e.libPaths, e.filename)
	if err != nil {
		return err
	}

	e.Node = node
	c.store[key] = *e
	return nil
}

// Remove removes an item from the node cache.
func (c *NodeCache) Remove(key string) error {
	return nil
}

// UpdateNodeCache updates the node cache using a file.
func UpdateNodeCache(path string, libPaths []string, cache *NodeCache) error {
	logrus.WithFields(logrus.Fields{
		"path":     path,
		"libPaths": strings.Join(libPaths, ",")}).
		Info("updating node cache")

	ic := NewImportCollector(libPaths)
	pathImports, err := ic.Collect(path, true)
	if err != nil {
		return err
	}

	logrus.Infof("(before) cache keys %s", strings.Join(cache.Keys(), ","))

	for _, pathImport := range pathImports {
		path, err := ImportPath(pathImport, libPaths)
		if err != nil {
			return err
		}

		importedFiles, err := ic.Collect(path, false)
		if err != nil {
			return err
		}

		ncds, err := collectNodeDependencies(path, importedFiles, libPaths)
		if err != nil {
			return errors.Wrap(err, "collecting import dependencies")
		}

		ne := NewNodeEntry(ncds, libPaths, pathImport)
		if err := cache.Set(pathImport, ne); err != nil {
			return err
		}
	}

	logrus.Infof("(after) cache keys %s", strings.Join(cache.Keys(), ","))

	return nil
}

func collectNodeDependencies(path string, names, libPaths []string) ([]NodeCacheDependency, error) {
	ncds := []NodeCacheDependency{}
	for _, importImport := range names {
		importPath, err := ImportPath(importImport, libPaths)
		if err != nil {
			return nil, errors.Wrap(err, "finding path for import in import")
		}

		fi, err := os.Stat(importPath)
		if err != nil {
			return nil, err
		}

		logrus.Info("building cache dependency")
		ncd := NodeCacheDependency{
			Name:      importImport,
			UpdatedAt: fi.ModTime(),
		}

		ncds = append(ncds, ncd)
	}

	return ncds, nil
}

// NodeBuilder builds ast.Node from source.
type NodeBuilder interface {
	Build(libPaths []string, name string) (ast.Node, error)
}

type nodeBuilder struct {
}

func (nb *nodeBuilder) Build(libPaths []string, name string) (ast.Node, error) {
	for _, libPath := range libPaths {
		sourcePath := filepath.Join(libPath, name)
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			continue
		}

		/* #nosec */
		source, err := ioutil.ReadFile(sourcePath)
		if err != nil {
			return nil, err
		}

		vm := jsonnet.MakeVM()
		importer := &jsonnet.FileImporter{
			JPaths: libPaths,
		}
		vm.Importer(importer)

		return vm.EvaluateToNode(sourcePath, string(source))
	}

	return nil, errors.Errorf("unable to find import %q", name)
}