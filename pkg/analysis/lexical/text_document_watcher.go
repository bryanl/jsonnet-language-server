package lexical

import (
	"context"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/jsonrpc2"
)

// TextDocumentWatcherConfig is configuration for TextDocumentWatcher.
type TextDocumentWatcherConfig interface {
	LocatableCache() *locate.LocatableCache
	Watch(string, config.DispatchFn) config.DispatchCancelFn
}

// TextDocumentWatcher watches text documents.
type TextDocumentWatcher struct {
	config TextDocumentWatcherConfig
	conn   *jsonrpc2.Conn
}

// NewTextDocumentWatcher creates an instance of NewTextDocumentWatcher.
func NewTextDocumentWatcher(c TextDocumentWatcherConfig) *TextDocumentWatcher {
	tdw := &TextDocumentWatcher{
		config: c,
	}

	c.Watch(config.TextDocumentUpdates, tdw.watch)

	return tdw
}

func (tdw *TextDocumentWatcher) SetConn(conn *jsonrpc2.Conn) {
	tdw.conn = conn
}

func (tdw *TextDocumentWatcher) watch(item interface{}) error {
	logger := logrus.WithField("component", "tdw")
	tdi, ok := item.(config.TextDocument)
	if !ok {
		return errors.Errorf("text document watcher can't handle %T", item)
	}

	logger.Infof("caching %s", tdi.URI())

	filename, err := uri.ToPath(tdi.URI())
	if err != nil {
		return err
	}

	r := strings.NewReader(tdi.String())

	done := make(chan bool, 1)
	diagCh := make(chan token.ParseDiagnostic, 1)

	diagnostics := make([]lsp.Diagnostic, 0)

	go func() {
		if diagCh == nil {
			close(done)
			return
		}

		for d := range diagCh {
			if tdw.conn != nil {
				r := position.FromJsonnetRange(d.Loc)

				diagnostic := lsp.Diagnostic{
					Range:    r.ToLSP(),
					Message:  d.Message,
					Severity: lsp.Error,
				}

				diagnostics = append(diagnostics, diagnostic)
			}
		}

		close(done)
	}()

	lv, err := newLocatableVisitor(filename, r, diagCh)
	if err != nil {
		logger.WithError(err).Info("creating visitor")
		// The document might not be parseable, but that's not a
		// error.
		return nil
	}

	logger.Info("running visitText")
	if err := lv.Visit(); err != nil {
		logger.WithError(err).Errorf("text document watcher visit nodes in %s",
			tdi.URI())
	}

	<-done

	logger.Info("sending diagnostics")
	response := &lsp.PublishDiagnosticsParams{
		URI:         tdi.URI(),
		Diagnostics: diagnostics,
	}

	ctx := context.Background()
	method := "textDocument/publishDiagnostics"
	if err := tdw.conn.Notify(ctx, method, response); err != nil {
		logrus.WithError(err).Error("sending diagnostics")
	}

	return nil
}
