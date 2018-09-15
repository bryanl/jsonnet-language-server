package server

import (
	"context"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	opentracing "github.com/opentracing/opentracing-go"
)

func textDocumentReferences(ctx context.Context, r *request, c *config.Config) (interface{}, error) {
	span := opentracing.SpanFromContext(ctx)
	ctx = opentracing.ContextWithSpan(ctx, span)

	var params lsp.ReferenceParams
	if err := r.Decode(&params); err != nil {
		return nil, err
	}

	doc, err := c.Text(ctx, params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	path, err := uri.ToPath(params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	pos := jpos.FromLSPPosition(params.Position)

	locations, err := token.Highlight(ctx, path, doc.String(), pos, c.NodeCache())
	if err != nil {
		return nil, err
	}

	var lspLocations []lsp.Location
	for _, l := range locations.Slice() {
		lspLocations = append(lspLocations, l.ToLSP())
	}

	return lspLocations, nil
}
