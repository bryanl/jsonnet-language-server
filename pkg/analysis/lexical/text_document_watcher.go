package lexical

import (
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/pkg/errors"
)

// TextDocumentWatcher watches text documents.
type TextDocumentWatcher struct {
	config *config.Config
	cache  *LocatableCache
}

// NewTextDocumentWatcher creates an instance of NewTextDocumentWatcher.
func NewTextDocumentWatcher(c *config.Config, cache *LocatableCache) *TextDocumentWatcher {
	tdw := &TextDocumentWatcher{
		config: c,
		cache:  cache,
	}

	c.Watch(config.CfgTextDocumentUpdates, tdw.watch)

	return tdw
}

func (tdw *TextDocumentWatcher) watch(item interface{}) error {
	tdi, ok := item.(lsp.TextDocumentItem)
	if !ok {
		return errors.Errorf("text document watcher can't handle %T", item)
	}

	filename, err := uri.ToPath(tdi.URI)
	if err != nil {
		return err
	}

	r := strings.NewReader(tdi.Text)

	_, err = newLocatableVisitor(filename, r, tdw.cache)
	if err != nil {
		return err
	}

	return nil
}
