package server

import (
	"context"
	"encoding/json"
	"net/url"
	"os"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical"
	"github.com/davecgh/go-spew/spew"

	"github.com/google/go-jsonnet/ast"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/go-langserver/pkg/lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func NewHandler(logger logrus.FieldLogger) jsonrpc2.Handler {
	return &lspHandler{
		logger: logger.WithField("component", "handler"),
	}
}

type lspHandler struct {
	logger logrus.FieldLogger
}

func (lh *lspHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	l := lh.logger.WithField("method", req.Method)

	switch req.Method {
	case "initialize":
		var ip lsp.InitializeParams
		if err := json.Unmarshal(*req.Params, &ip); err != nil {
			l.WithError(err).Error("invalid payload")
			return
		}

		lh.logger.WithFields(logrus.Fields{
			"workspace": ip.RootPath,
			"id":        req.ID,
		}).Info("initialzing")

		ir := &lsp.InitializeResult{
			Capabilities: lsp.ServerCapabilities{
				HoverProvider: true,
			},
		}

		if err := conn.Reply(ctx, req.ID, ir); err != nil {
			lh.logger.WithError(err).Error("reply error")
		}

	case "textDocument/hover":
		var tdpp lsp.TextDocumentPositionParams
		if err := json.Unmarshal(*req.Params, &tdpp); err != nil {
			l.WithError(err).Error("invalid payload")
			return
		}

		l.WithField("pos", tdpp.Position).Info("handling hover")

		u, err := url.Parse(string(tdpp.TextDocument.URI))
		if err != nil {
			l.WithError(err).Error(err)
			return
		}

		if u.Scheme != "file" {
			l.Error("invalid file schema")
			return
		}

		f, err := os.Open(u.Path)
		if err != nil {
			l.WithError(err).Error("open file")
			return
		}

		loc := ast.Location{
			Line:   tdpp.Position.Line + 1,
			Column: tdpp.Position.Character,
		}

		v, err := lexical.NewCursorVisitor(u.Path, f, loc)
		if err != nil {
			l.WithError(err).Error("create cursor visitor")
			return
		}

		if err = v.Visit(); err != nil {
			l.WithError(err).Error("visit nodes")
			return
		}

		locatable, err := v.TokenAtPosition()
		if err != nil {
			l.WithError(err).Error("find token at position")
			return
		}

		spew.Sdump(locatable)

		hover := &lsp.Hover{
			Contents: []lsp.MarkedString{
				{
					Language: "markdown",
					Value:    spew.Sdump(locatable),
				},
			},
			Range: lsp.Range{
				Start: lsp.Position{Line: locatable.Loc.Begin.Line - 1, Character: locatable.Loc.Begin.Column - 1},
				End:   lsp.Position{Line: locatable.Loc.End.Line - 1, Character: locatable.Loc.End.Column - 1},
			},
		}

		if err := conn.Reply(ctx, req.ID, hover); err != nil {
			lh.logger.WithError(err).Error("reply error")
		}
	default:
		lh.logger.WithFields(logrus.Fields{
			"method": req.Method,
			"id":     req.ID,
			"params": string(*req.Params),
		}).Info("unknown message type")

	}
}
