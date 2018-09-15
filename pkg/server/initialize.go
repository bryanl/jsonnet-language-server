package server

import (
	"context"
	"path/filepath"

	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
)

func initialize(ctx context.Context, r *request, c *config.Config) (interface{}, error) {
	span := opentracing.SpanFromContext(ctx)
	ctx = opentracing.ContextWithSpan(ctx, span)

	var ip lsp.InitializeParams
	if err := r.Decode(&ip); err != nil {
		return nil, err
	}

	fn := func(ctx context.Context, v interface{}) error {
		span := opentracing.SpanFromContext(ctx)
		ctx = opentracing.ContextWithSpan(ctx, span)

		// When lib paths are updated, tell the client to send
		// watch updates for all the lib paths.

		paths, ok := v.([]string)
		if !ok {
			span.LogFields(
				log.Error(errors.New("lib paths are not []string")),
			)
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

		if _, err := r.RegisterCapability(ctx, "workspace/didChangeWatchedFiles", options); err != nil {
			span.LogFields(log.Error(err))
		}

		return nil
	}

	c.Watch(config.JsonnetLibPaths, fn)

	update, ok := ip.InitializationOptions.(map[string]interface{})
	if !ok {
		return nil, errors.New("initialization options are incorrect type")
	}

	if err := c.UpdateClientConfiguration(ctx, update); err != nil {
		return nil, err
	}

	span.LogFields(
		log.String("workspace", ip.RootPath),
		log.String("config", c.String()),
	)

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
