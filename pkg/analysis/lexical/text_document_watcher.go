package lexical

import (
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// TextDocumentWatcher watches text documents.
type TextDocumentWatcher struct {
	config *config.Config
}

// NewTextDocumentWatcher creates an instance of NewTextDocumentWatcher.
func NewTextDocumentWatcher(c *config.Config) *TextDocumentWatcher {
	tdw := &TextDocumentWatcher{
		config: c,
	}

	c.Watch(config.TextDocumentUpdates, tdw.watch)

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

	lv, err := newLocatableVisitor(filename, r)
	if err != nil {
		return err
	}

	logrus.Info("running visitText")
	if err := lv.Visit(); err != nil {
		return err
	}

	locatableCache := tdw.config.LocatableCache()
	return locatableCache.Store(filename, lv.Locatables())
}
