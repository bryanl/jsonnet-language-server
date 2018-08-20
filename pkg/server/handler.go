package server

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime/debug"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/jsonrpc2"
)

type operation func(*request, *Config) (interface{}, error)

var operations = map[string]operation{
	"initialize":                initialize,
	"textDocument/completion":   textDocumentCompletion,
	"textDocument/didChange":    textDocumentDidChange,
	"textDocument/didClose":     textDocumentDidClose,
	"textDocument/didOpen":      textDocumentDidOpen,
	"textDocument/didSave":      textDocumentDidSave,
	"textDocument/hover":        textDocumentHover,
	"updateClientConfiguration": updateClientConfiguration,
}

type lspHandler struct {
	logger    logrus.FieldLogger
	config    *Config
	decoder   *requestDecoder
	nodeCache *locate.NodeCache
}

// NewHandler creates a handler to handle rpc commands.
func NewHandler(logger logrus.FieldLogger) jsonrpc2.Handler {
	config := NewConfig()

	return &lspHandler{
		logger:    logger.WithField("component", "handler"),
		decoder:   &requestDecoder{},
		config:    config,
		nodeCache: locate.NewNodeCache(),
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

func (r *request) RegisterCapability(method string, options interface{}) (string, error) {
	id := uuid.Must(uuid.NewV4())

	registrations := &lsp.RegistrationParams{
		Registrations: []lsp.Registration{
			{
				ID:              id.String(),
				Method:          method,
				RegisterOptions: options,
			},
		},
	}

	var result interface{}

	if err := r.conn.Call(r.ctx, "client/registerCapability", registrations, result); err != nil {
		return "", err
	}

	return id.String(), nil
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

	fn := func(v interface{}) {
		// When lib paths are updated, tell the client to send
		// watch updates for all the lib paths.

		paths, ok := v.([]string)
		if !ok {
			r.log().Error("lib paths are not []string")
		}

		options := &lsp.DidChangeWatchedFilesRegistrationOptions{
			Watchers: make([]lsp.FileSystemWatcher, 0),
		}

		for _, path := range paths {
			path = filepath.Clean(path)
			for _, ext := range []string{"libsonnet", "jsonnet"} {
				watcher := lsp.FileSystemWatcher{
					GlobPattern: filepath.Join(path, "*."+ext),
					Kind:        lsp.WatchKindChange + lsp.WatchKindCreate + lsp.WatchKindDelete,
				}

				options.Watchers = append(options.Watchers, watcher)
			}
		}

		if _, err := r.RegisterCapability("workspace/didChangeWatchedFiles", options); err != nil {
			r.log().WithError(err).Error("registering file watchers")
		}
	}

	c.Watch(CfgJsonnetLibPaths, fn)

	update, ok := ip.InitializationOptions.(map[string]interface{})
	if !ok {
		return nil, errors.New("initialization options are incorrect type")
	}

	if err := c.update(update); err != nil {
		return nil, err
	}

	r.log().WithFields(logrus.Fields{
		"workspace": ip.RootPath,
		"config":    c.String(),
	}).Info("initializing")

	response := &lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			CompletionProvider: &lsp.CompletionOptions{
				ResolveProvider: true,
			},
			HoverProvider:    true,
			TextDocumentSync: lsp.TDSKFull,
		},
	}

	return response, nil
}

func textDocumentCompletion(r *request, c *Config) (interface{}, error) {
	var rp lsp.ReferenceParams
	if err := r.Decode(&rp); err != nil {
		return nil, err
	}

	cmpl := newComplete(rp, c)
	response, err := cmpl.handle()
	if err != nil {
		logrus.WithError(err).Error("completion erred")
		return nil, err
	}

	return response, nil
}

func textDocumentHover(r *request, c *Config) (interface{}, error) {
	var tdpp lsp.TextDocumentPositionParams
	if err := r.Decode(&tdpp); err != nil {
		return nil, err
	}

	h := newHover(tdpp, c)
	return h.handle()
}

func updateClientConfiguration(r *request, c *Config) (interface{}, error) {
	var update map[string]interface{}
	if err := r.Decode(&update); err != nil {
		return nil, err
	}

	if err := c.update(update); err != nil {
		if msgErr := showMessage(r, lsp.MTError, err.Error()); msgErr != nil {
			r.log().WithError(msgErr).Error("sending message")
		}

		return nil, err
	}

	return nil, nil
}

func updateNodeCache(r *request, c *Config, uri string) {
	path, err := uriToPath(uri)
	if err != nil {
		r.log().WithError(err).Error("converting URI to path")
		return
	}

	if err := locate.UpdateNodeCache(path, c.JsonnetLibPaths(), c.NodeCache()); err != nil {
		r.log().WithError(err).
			WithField("uri", path).
			Error("updating node cache")
	}
}

func textDocumentDidOpen(r *request, c *Config) (interface{}, error) {
	var dotdp lsp.DidOpenTextDocumentParams
	if err := r.Decode(&dotdp); err != nil {
		return nil, err
	}

	r.log().WithField("uri", dotdp.TextDocument.URI).Info("opened file")

	go updateNodeCache(r, c, dotdp.TextDocument.URI)

	return nil, nil
}

func textDocumentDidSave(r *request, c *Config) (interface{}, error) {
	var dotdp lsp.DidOpenTextDocumentParams
	if err := r.Decode(&dotdp); err != nil {
		return nil, err
	}

	r.log().WithField("uri", dotdp.TextDocument.URI).Info("saved file")

	go updateNodeCache(r, c, dotdp.TextDocument.URI)

	return nil, nil
}

func textDocumentDidClose(r *request, c *Config) (interface{}, error) {
	var dotdp lsp.DidOpenTextDocumentParams
	if err := r.Decode(&dotdp); err != nil {
		return nil, err
	}

	r.log().WithField("uri", dotdp.TextDocument.URI).Info("closed file")

	go updateNodeCache(r, c, dotdp.TextDocument.URI)

	return nil, nil
}

func textDocumentDidChange(r *request, c *Config) (interface{}, error) {
	var dotdp lsp.DidOpenTextDocumentParams
	if err := r.Decode(&dotdp); err != nil {
		return nil, err
	}

	if err := c.storeTextDocumentItem(dotdp.TextDocument); err != nil {
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
