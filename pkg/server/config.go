package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
)

const (
	// CfgJsonnetLibPaths are jsonnet lib paths.
	CfgJsonnetLibPaths = "jsonnet.libPaths"
)

// Config is configuration setting for the server.
type Config struct {
	// Files holds the files as sent from the client.
	Files map[string]lsp.TextDocumentItem

	// JsonnetLibPaths are jsonnet lib paths.
	JsonnetLibPaths []string

	// NodeCache holds the node cache.
	NodeCache *locate.NodeCache

	dispatchers map[string]*Dispatcher
}

// NewConfig creates an instance of Config.
func NewConfig() *Config {
	return &Config{
		Files:           make(map[string]lsp.TextDocumentItem),
		JsonnetLibPaths: make([]string, 0),
		NodeCache:       locate.NewNodeCache(),

		dispatchers: map[string]*Dispatcher{},
	}
}

// UpdateFile updates the local file cache.
func (c *Config) UpdateFile(tdi lsp.TextDocumentItem) error {
	c.Files[tdi.URI] = tdi
	return nil
}

// Text retrieves text from our local cache or from the file system.
func (c *Config) Text(uri string) (string, error) {
	text, ok := c.Files[uri]
	if ok {
		logrus.Info("returning text from cache")
		return text.Text, nil
	}
	logrus.Info("returning text from disk")
	path, err := uriToPath(uri)
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
func (c *Config) Watch(k string, fn func(interface{})) func() {
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

// Update updates the configuration.
func (c *Config) Update(update map[string]interface{}) error {
	for k, v := range update {
		switch k {
		case CfgJsonnetLibPaths:
			paths, err := interfaceToStrings(v)
			if err != nil {
				return errors.Wrapf(err, "setting %q", CfgJsonnetLibPaths)
			}

			c.JsonnetLibPaths = paths
			c.dispatch(CfgJsonnetLibPaths, paths)
		default:
			return errors.Errorf("setting %q is unknown to the jsonnet language server", k)
		}
	}
	return nil
}

func (c *Config) String() string {
	data, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Sprintf("marshaling config to JSON: %v", err))
	}
	return string(data)
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
