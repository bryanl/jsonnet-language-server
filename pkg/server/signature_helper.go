package server

import (
	"context"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	opentracing "github.com/opentracing/opentracing-go"
)

func textDocumentSignatureHelper(ctx context.Context, r *request, c *config.Config) (interface{}, error) {
	span := opentracing.SpanFromContext(ctx)
	ctx = opentracing.ContextWithSpan(ctx, span)

	var params lsp.TextDocumentPositionParams
	if err := r.Decode(&params); err != nil {
		return nil, err
	}

	text, err := c.Text(ctx, params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	pos := jpos.FromLSPPosition(params.Position)

	sr, err := token.SignatureHelper(text.String(), pos, c.NodeCache())
	if err != nil {
		return nil, err
	}

	si := lsp.SignatureInformation{
		Label:      sr.Label,
		Parameters: []lsp.ParameterInformation{},
	}

	for _, param := range sr.Parameters {
		si.Parameters = append(si.Parameters, lsp.ParameterInformation{Label: param})
	}

	response := &lsp.SignatureHelp{
		Signatures: []lsp.SignatureInformation{si},
	}

	return response, nil
}
