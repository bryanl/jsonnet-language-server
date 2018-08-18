package server

import (
	"context"
	"encoding/json"
	"runtime/debug"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/go-langserver/pkg/lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type operation func(*request, *Config) (interface{}, error)

var operations = map[string]operation{
	"textDocument/hover":        textDocumentHover,
	"initialize":                initialize,
	"updateClientConfiguration": updateClientConfiguration,
}

// NewHandler creates a handler to handle rpc commands.
func NewHandler(logger logrus.FieldLogger) jsonrpc2.Handler {
	return &lspHandler{
		logger:  logger.WithField("component", "handler"),
		decoder: &requestDecoder{},
		config:  NewConfig(),
	}
}

type request struct {
	ctx     context.Context
	conn    *jsonrpc2.Conn
	req     *jsonrpc2.Request
	logger  logrus.FieldLogger
	decoder *requestDecoder
}

func (r *request) log() logrus.FieldLogger {
	return r.logger.WithFields(logrus.Fields{
		"method": r.req.Method,
		"id":     r.req.ID.String()})
}

func (r *request) Decode(v interface{}) error {
	return r.decoder.Decode(r.req, v)
}

type lspHandler struct {
	logger  logrus.FieldLogger
	config  *Config
	decoder *requestDecoder
}

func (lh *lspHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	l := lh.logger.WithFields(logrus.Fields{
		"method": req.Method,
		"id":     req.ID.String()})

	r := &request{
		ctx:    ctx,
		conn:   conn,
		req:    req,
		logger: l,
	}

	defer func() {
		if r := recover(); r != nil {
			l.Errorf("(CRASH) %v: %s", r, debug.Stack())
		}
	}()

	fn, ok := operations[req.Method]
	if !ok {
		l.WithFields(logrus.Fields{
			"params": string(*req.Params),
		}).Info("unknown message type")
		return
	}

	response, err := fn(r, lh.config)
	if err != nil {
		msg := &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInternalError,
			Message: err.Error(),
		}
		if replyErr := conn.ReplyWithError(ctx, req.ID, msg); err != nil {
			l.WithError(replyErr).Error("replying with error")
		}
		return
	}

	if err := conn.Reply(ctx, req.ID, response); err != nil {
		l.WithError(err).Error("reply")
	}
}

func initialize(r *request, c *Config) (interface{}, error) {
	var ip lsp.InitializeParams
	if err := r.Decode(&ip); err != nil {
		return nil, err
	}

	update, ok := ip.InitializationOptions.(map[string]interface{})
	if !ok {
		return nil, errors.New("initialization options are incorrect type")
	}

	if err := c.Update(update); err != nil {
		return nil, err
	}

	r.log().WithFields(logrus.Fields{
		"workspace": ip.RootPath,
		"config":    c.String(),
	}).Info("initializing")

	response := &lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			HoverProvider: true,
		},
	}

	return response, nil
}

func textDocumentHover(r *request, c *Config) (interface{}, error) {
	var tdpp lsp.TextDocumentPositionParams
	if err := r.Decode(&tdpp); err != nil {
		return nil, err
	}

	h := newHover(tdpp)
	return h.handle()
}

func updateClientConfiguration(r *request, c *Config) (interface{}, error) {
	var update map[string]interface{}
	if err := r.Decode(&update); err != nil {
		return nil, err
	}

	if err := c.Update(update); err != nil {
		if msgErr := showMessage(r, lsp.MTError, err.Error()); msgErr != nil {
			r.log().WithError(msgErr).Error("sending message")
		}

		return nil, err
	}

	return nil, nil
}

func showMessage(r *request, mt lsp.MessageType, message string) error {
	smp := &lsp.ShowMessageParams{
		Type:    int(mt),
		Message: message,
	}

	return r.conn.Notify(r.ctx, "window/showMessage", smp)
}

type requestDecoder struct {
}

func (rd *requestDecoder) Decode(req *jsonrpc2.Request, v interface{}) error {
	if err := json.Unmarshal(*req.Params, v); err != nil {
		return errors.Wrap(err, "invalid payload")
	}

	return nil
}
