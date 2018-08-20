package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// JsonnetLibPaths are jsonnet lib paths.
	JsonnetLibPaths = "jsonnet.libPaths"

	// TextDocumentUpdates are text document updates.
	TextDocumentUpdates = "textDocument.update"
)

// TextDocument is a document's text and and metadata.
type TextDocument struct {
	URI        string
	LanguageID string
	Version    int
	Text       string
}

// Config is configuration setting for the server.
type Config struct {
	textDocuments   map[string]TextDocument
	jsonnetLibPaths []string
	nodeCache       *locate.NodeCache
	locatableCache  *locate.LocatableCache
	dispatchers     map[string]*Dispatcher
}

// New creates an instance of Config.
func New() *Config {
	return &Config{
		textDocuments:   make(map[string]TextDocument),
		jsonnetLibPaths: make([]string, 0),
		nodeCache:       locate.NewNodeCache(),
		locatableCache:  locate.NewLocatableCache(),
		dispatchers:     map[string]*Dispatcher{},
	}
}

// LocatableCache returns the locatable cache.
func (c *Config) LocatableCache() *locate.LocatableCache {
	return c.locatableCache
}

// NodeCache returns the node cache.
func (c *Config) NodeCache() *locate.NodeCache {
	return c.nodeCache
}

// JsonnetLibPaths returns Jsonnet lib paths.
func (c *Config) JsonnetLibPaths() []string {
	return c.jsonnetLibPaths
}

// StoreTextDocumentItem stores a text document item.
func (c *Config) StoreTextDocumentItem(td TextDocument) error {
	oldDoc, ok := c.textDocuments[td.URI]
	if !ok {
		oldDoc = td
	}

	oldDoc.Text = td.Text
	oldDoc.Version = td.Version

	c.textDocuments[td.URI] = td
	c.dispatch(TextDocumentUpdates, td)
	return nil
}

// UpdateTextDocumentItem updates a text document item with a change event.
func (c *Config) UpdateTextDocumentItem(dctdp lsp.DidChangeTextDocumentParams) error {
	// The language server is configured to request for full content changes,
	// so the text in the change event is a full document.

	td := TextDocument{
		Text:    dctdp.ContentChanges[0].Text,
		URI:     dctdp.TextDocument.URI,
		Version: dctdp.TextDocument.Version,
	}

	return c.StoreTextDocumentItem(td)
}

// Text retrieves text from our local cache or from the file system.
func (c *Config) Text(uriStr string) (string, error) {
	text, ok := c.textDocuments[uriStr]
	if ok {
		logrus.Info("returning text from cache")
		return text.Text, nil
	}
	logrus.Info("returning text from disk")
	path, err := uri.ToPath(uriStr)
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Watch will call `fn`` when key `k` is updated. It returns a
// cancel function.
func (c *Config) Watch(k string, fn DispatchFn) DispatchCancelFn {
	d := c.dispatcher(k)
	return d.Watch(fn)
}

func (c *Config) dispatcher(k string) *Dispatcher {
	d, ok := c.dispatchers[k]
	if !ok {
		d = NewDispatcher()
		c.dispatchers[k] = d
	}

	return d
}

func (c *Config) dispatch(k string, msg interface{}) {
	d := c.dispatcher(k)
	d.Dispatch(msg)
}

// UpdateClientConfiguration updates the configuration.
func (c *Config) UpdateClientConfiguration(update map[string]interface{}) error {
	for k, v := range update {
		switch k {
		case JsonnetLibPaths:
			paths, err := interfaceToStrings(v)
			if err != nil {
				return errors.Wrapf(err, "setting %q", JsonnetLibPaths)
			}

			c.jsonnetLibPaths = paths
			c.dispatch(JsonnetLibPaths, paths)
		default:
			return errors.Errorf("setting %q is unknown to the jsonnet language server", k)
		}
	}
	return nil
}

func (c *Config) String() string {
	data, err := c.MarshalJSON()
	if err != nil {
		panic(fmt.Sprintf("marshaling config to JSON: %v", err))
	}
	return string(data)
}

type configMarshaled struct {
	JsonnetLibPaths []string
}

// MarshalJSON marshals a config to JSON bytes.
func (c *Config) MarshalJSON() ([]byte, error) {
	cm := configMarshaled{
		JsonnetLibPaths: c.JsonnetLibPaths(),
	}

	return json.Marshal(&cm)
}

// getTextDocumentItem returns a text document item by URI.
func (c *Config) getTextDocumentItem(uri string) (TextDocument, error) {
	item, ok := c.textDocuments[uri]
	if !ok {
		return TextDocument{}, errors.Errorf("uri %q does not exist", uri)
	}

	return item, nil
}

func interfaceToStrings(v interface{}) ([]string, error) {
	switch v := v.(type) {
	case []interface{}:
		var out []string
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, errors.Errorf("item was not a string")
			}

			out = append(out, str)
		}

		return out, nil
	case []string:
		return v, nil
	default:
		return nil, errors.Errorf("unable to convert %T to array of strings", v)
	}
}