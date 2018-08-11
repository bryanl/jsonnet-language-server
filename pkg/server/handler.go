package server

import (
	"context"
	"encoding/json"

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
	l := lh.logger.WithFields(logrus.Fields{
		"method": req.Method,
		"id":     req.ID.String()})

	var response interface{}

	switch req.Method {
	case "initialize":
		var ip lsp.InitializeParams
		if err := json.Unmarshal(*req.Params, &ip); err != nil {
			l.WithError(err).Error("invalid payload")
			return
		}

		l.WithFields(logrus.Fields{
			"workspace": ip.RootPath,
		}).Info("initialzing")

		response = &lsp.InitializeResult{
			Capabilities: lsp.ServerCapabilities{
				HoverProvider: true,
			},
		}
	case "textDocument/hover":
		var tdpp lsp.TextDocumentPositionParams
		if err := json.Unmarshal(*req.Params, &tdpp); err != nil {
			lh.logger.WithError(err).Error("invalid payload")
		}

		var err error
		h := newHover(tdpp)
		response, err = h.handle()
		if err != nil {
			l.WithError(err).Error("handle hover")
			return
		}
	default:
		lh.logger.WithFields(logrus.Fields{
			"method": req.Method,
			"id":     req.ID,
			"params": string(*req.Params),
		}).Info("unknown message type")

	}

	if err := conn.Reply(ctx, req.ID, response); err != nil {
		l.WithError(err).Error("reply error")
	}
}
