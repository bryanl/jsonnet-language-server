package locate

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/go-jsonnet/ast"
)

// NodeCacheMissErr is an error for a cache miss.
type NodeCacheMissErr struct {
	key string
}

func (e *NodeCacheMissErr) Error() string {
	return fmt.Sprintf("%q did not exist", e.key)
}

// NodeEntry is an entry in the NodeCache.
type NodeEntry struct {
	Node         ast.Node
	Dependencies []string
	UpdatedAt    *time.Time
}

// NodeCache is a cache for nodes.
type NodeCache struct {
	store map[string]NodeEntry

	mu sync.Mutex
}

// NewNodeCache creates an instance of NodeCache.
func NewNodeCache() *NodeCache {
	c := &NodeCache{
		store: make(map[string]NodeEntry),
	}

	return c
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

	c.store[key] = *e

	return nil
}
