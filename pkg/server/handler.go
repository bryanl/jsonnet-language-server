package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"

	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/jsonrpc2"
)

type operation func(*request, *config.Config) (interface{}, error)

var operations = map[string]operation{
	"completionItem/resolve":         completionItemResolve,
	"initialize":                     initialize,
	"textDocument/completion":        textDocumentCompletion,
	"textDocument/didChange":         textDocumentDidChange,
	"textDocument/didClose":          textDocumentDidClose,
	"textDocument/didOpen":           textDocumentDidOpen,
	"textDocument/didSave":           textDocumentDidSave,
	"textDocument/documentHighlight": textDocumentHighlight,
	"textDocument/documentSymbol":    textDocumentSymbol,
	"textDocument/hover":             textDocumentHover,
	"textDocument/references":        textDocumentReferences,
	"textDocument/signatureHelp":     textDocumentSignatureHelper,
	"updateClientConfiguration":      updateClientConfiguration,
}

// Handler is a JSON RPC Handler
type Handler struct {
	logger              logrus.FieldLogger
	config              *config.Config
	decoder             *requestDecoder
	nodeCache           *token.NodeCache
	textDocumentWatcher *lexical.TextDocumentWatcher
	conn                *jsonrpc2.Conn
}

var _ jsonrpc2.Handler = (*Handler)(nil)

// NewHandler creates a handler to handle rpc commands.
func NewHandler(logger logrus.FieldLogger) *Handler {
	c := config.New()
	nodeCache := token.NewNodeCache()

	tdw := lexical.NewTextDocumentWatcher(c, lexical.NewPerformDiagnostics())

	return &Handler{
		logger:              logger.WithField("component", "handler"),
		decoder:             &requestDecoder{},
		config:              c,
		nodeCache:           nodeCache,
		textDocumentWatcher: tdw,
	}
}

// SetConn sets the RPC connection for the handler.
func (h *Handler) SetConn(conn *jsonrpc2.Conn) {
	h.conn = conn
	h.textDocumentWatcher.SetConn(conn)
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
	id := uuid.NewV4()

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

// Handle handles a JSON RPC connection.
func (lh *Handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
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
			log.Printf("(CRASH) %v: %s", r, debug.Stack())
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

func initialize(r *request, c *config.Config) (interface{}, error) {
	var ip lsp.InitializeParams
	if err := r.Decode(&ip); err != nil {
		return nil, err
	}

	fn := func(v interface{}) error {
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

		return nil
	}

	c.Watch(config.JsonnetLibPaths, fn)

	update, ok := ip.InitializationOptions.(map[string]interface{})
	if !ok {
		return nil, errors.New("initialization options are incorrect type")
	}

	if err := c.UpdateClientConfiguration(update); err != nil {
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
			DocumentSymbolProvider:    true,
			DocumentHighlightProvider: true,
			HoverProvider:             true,
			ReferencesProvider:        true,
			SignatureHelpProvider: &lsp.SignatureHelpOptions{
				TriggerCharacters: []string{"("},
			},
			TextDocumentSync: lsp.TDSKFull,
		},
	}

	return response, nil
}

func completionItemResolve(r *request, c *config.Config) (interface{}, error) {
	var ci lsp.CompletionItem
	if err := r.Decode(&ci); err != nil {
		return nil, err
	}

	// TODO figure out what do do here. This might not be needed and
	// it can drop.
	return nil, nil
}

func textDocumentCompletion(r *request, c *config.Config) (interface{}, error) {
	var rp lsp.ReferenceParams
	if err := r.Decode(&rp); err != nil {
		return nil, err
	}

	cmpl, err := newComplete(rp, c)
	if err != nil {
		return nil, err
	}

	response, err := cmpl.handle()
	if err != nil {
		logrus.WithError(err).Error("completion erred")
		return nil, err
	}

	return response, nil
}

func textDocumentHover(r *request, c *config.Config) (interface{}, error) {
	var tdpp lsp.TextDocumentPositionParams
	if err := r.Decode(&tdpp); err != nil {
		return nil, err
	}

	h, err := newHover(tdpp, c)
	if err != nil {
		return nil, err
	}

	return h.handle()
}

func updateClientConfiguration(r *request, c *config.Config) (interface{}, error) {
	var update map[string]interface{}
	if err := r.Decode(&update); err != nil {
		return nil, err
	}

	if err := c.UpdateClientConfiguration(update); err != nil {
		if msgErr := showMessage(r, lsp.MTError, err.Error()); msgErr != nil {
			r.log().WithError(msgErr).Error("sending message")
		}

		return nil, err
	}

	return nil, nil
}

func updateNodeCache(r *request, c *config.Config, uriStr string) {
	path, err := uri.ToPath(uriStr)
	if err != nil {
		r.log().WithError(err).Error("converting URI to path")
		return
	}

	// do notification stuff here

	done := make(chan bool, 1)
	errCh := make(chan error, 1)

	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	go func() {
		err := token.UpdateNodeCache(path, c.JsonnetLibPaths(), c.NodeCache())
		if err != nil {
			errCh <- err
			return
		}

		done <- true
	}()

	sentNotif := false

	_, file := filepath.Split(path)

	for {
		select {
		case err := <-errCh:
			r.log().WithError(err).
				WithField("uri", path).
				Error("updating node cache")
			return
		case <-done:
			if sentNotif {
				msg := fmt.Sprintf("Import processing for %q is complete", file)
				_ = showMessage(r, lsp.MTWarning, msg)
				logrus.Info("cancel notification")
			}
			return
		case <-timer.C:
			msg := fmt.Sprintf("Import processing for %q is running", file)
			_ = showMessage(r, lsp.MTWarning, msg)
			logrus.Info("send notification")
			sentNotif = true
		}
	}

}

func closeFile(r *request, c *config.Config, uriStr string) {
	path, err := uri.ToPath(uriStr)
	if err != nil {
		r.log().WithError(err).Error("converting URI to path")
		return
	}

	nodeCache := c.NodeCache()
	if err := nodeCache.Remove(path); err != nil {
		r.log().WithError(err).
			WithField("uri", path).
			Error("closing file")
	}
}

func textDocumentDidOpen(r *request, c *config.Config) (interface{}, error) {
	var dotdp lsp.DidOpenTextDocumentParams
	if err := r.Decode(&dotdp); err != nil {
		return nil, err
	}

	r.log().WithField("uri", dotdp.TextDocument.URI).Info("opened file")

	td := config.NewTextDocumentFromItem(dotdp.TextDocument)
	if err := c.StoreTextDocumentItem(td); err != nil {
		return nil, err
	}

	go updateNodeCache(r, c, dotdp.TextDocument.URI)

	return nil, nil
}

func textDocumentDidSave(r *request, c *config.Config) (interface{}, error) {
	var dotdp lsp.DidOpenTextDocumentParams
	if err := r.Decode(&dotdp); err != nil {
		return nil, err
	}

	r.log().WithField("uri", dotdp.TextDocument.URI).Info("saved file")

	go updateNodeCache(r, c, dotdp.TextDocument.URI)

	return nil, nil
}

func textDocumentDidClose(r *request, c *config.Config) (interface{}, error) {
	var params lsp.DidCloseTextDocumentParams
	if err := r.Decode(&params); err != nil {
		return nil, err
	}

	r.log().WithField("uri", params.TextDocument.URI).Info("closed file")

	go closeFile(r, c, params.TextDocument.URI)

	return nil, nil
}

func textDocumentDidChange(r *request, c *config.Config) (interface{}, error) {
	var dctdp lsp.DidChangeTextDocumentParams
	if err := r.Decode(&dctdp); err != nil {
		return nil, err
	}

	if err := c.UpdateTextDocumentItem(dctdp); err != nil {
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
