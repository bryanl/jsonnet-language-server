package server

import (
	"context"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
)

func textDocumentSymbol(ctx context.Context, r *request, c *config.Config) (interface{}, error) {
	var params lsp.DocumentSymbolParams
	if err := r.Decode(&params); err != nil {
		return nil, err
	}

	doc, err := c.Text(params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	symbols, err := token.Symbols(doc.String())
	if err != nil {
		return nil, err
	}

	var response []lsp.DocumentSymbol

	for _, symbol := range symbols {
		enclosingRange := symbol.Range()
		selectionRange := symbol.SelectionRange()

		ds := lsp.DocumentSymbol{
			Name:           symbol.Name(),
			Detail:         symbol.Detail(),
			Kind:           symbol.Kind(),
			Deprecated:     symbol.IsDeprecated(),
			Range:          enclosingRange.ToLSP(),
			SelectionRange: selectionRange.ToLSP(),
			Children:       make([]lsp.DocumentSymbol, 0),
		}

		response = append(response, ds)
	}

	return response, nil
}
