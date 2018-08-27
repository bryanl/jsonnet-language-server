package lexical

import (
	"context"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/sirupsen/logrus"
)

// DocumentProcessor processes TextDocument.
type DocumentProcessor interface {
	Process(td config.TextDocument, conn RPCConn) error
}

// PerformDiagnostics performs diagnostics on a text document and sends results
// to the client.
type PerformDiagnostics struct{}

var _ DocumentProcessor = (*PerformDiagnostics)(nil)

// NewPerformDiagnostics creates an instance of PerformDiagnostics.
func NewPerformDiagnostics() *PerformDiagnostics {
	return &PerformDiagnostics{}
}

// Process runs the diagnositics.
func (p *PerformDiagnostics) Process(td config.TextDocument, conn RPCConn) error {
	logger := logrus.WithField("component", "perform-diagnostics")

	logger.Infof("caching %s", td.URI())

	filename, err := uri.ToPath(td.URI())
	if err != nil {
		return err
	}

	r := strings.NewReader(td.String())

	done := make(chan bool, 1)
	diagCh := make(chan token.ParseDiagnostic, 1)

	diagnostics := make([]lsp.Diagnostic, 0)

	go func() {
		if diagCh == nil {
			close(done)
			return
		}

		for d := range diagCh {
			if conn != nil {
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

	_, err = NewNodeVisitor(filename, r, true, parseDiagOpt(diagCh))
	if err != nil {
		logger.WithError(err).Info("creating visitor")
		return err
	}

	<-done

	if conn != nil {
		logger.Info("sending diagnostics")
		response := &lsp.PublishDiagnosticsParams{
			URI:         td.URI(),
			Diagnostics: diagnostics,
		}

		ctx := context.Background()
		method := "textDocument/publishDiagnostics"
		if err := conn.Notify(ctx, method, response); err != nil {
			logger.WithError(err).Error("sending diagnostics")
		}

	}

	return nil
}
