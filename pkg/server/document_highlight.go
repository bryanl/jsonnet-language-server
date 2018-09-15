package server

import (
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
)

func textDocumentHighlight(r *request, c *config.Config) (interface{}, error) {
	var params lsp.TextDocumentPositionParams
	if err := r.Decode(&params); err != nil {
		return nil, err
	}

	doc, err := c.Text(params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	path, err := uri.ToPath(params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	pos := jpos.FromLSPPosition(params.Position)

	locations, err := token.Highlight(path, doc.String(), pos, c.NodeCache())
	if err != nil {
		return nil, err
	}

	var highlights []lsp.DocumentHighlight

	for _, location := range locations.Slice() {
		r := location.Range()
		dh := lsp.DocumentHighlight{
			Range: r.ToLSP(),
		}
		highlights = append(highlights, dh)
	}

	return highlights, nil
}
