package lexical

import (
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"

	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// TextDocumentWatcherConfig is configuration for TextDocumentWatcher.
type TextDocumentWatcherConfig interface {
	LocatableCache() *locate.LocatableCache
	Watch(string, config.DispatchFn) config.DispatchCancelFn
}

// TextDocumentWatcher watches text documents.
type TextDocumentWatcher struct {
	config TextDocumentWatcherConfig
}

// NewTextDocumentWatcher creates an instance of NewTextDocumentWatcher.
func NewTextDocumentWatcher(c TextDocumentWatcherConfig) *TextDocumentWatcher {
	tdw := &TextDocumentWatcher{
		config: c,
	}

	c.Watch(config.TextDocumentUpdates, tdw.watch)

	return tdw
}

func (tdw *TextDocumentWatcher) watch(item interface{}) error {
	tdi, ok := item.(config.TextDocument)
	if !ok {
		return errors.Errorf("text document watcher can't handle %T", item)
	}

	filename, err := uri.ToPath(tdi.URI())
	if err != nil {
		return err
	}

	r := strings.NewReader(tdi.String())

	lv, err := newLocatableVisitor(filename, r)
	if err != nil {
		// The document might not be parseable, but that's not a
		// error.
		return nil
	}

	logrus.Info("running visitText")
	if err := lv.Visit(); err != nil {
		return errors.Wrap(err, "visiting nodes")
	}

	locatableCache := tdw.config.LocatableCache()
	return locatableCache.Store(filename, lv.Locatables())
}
