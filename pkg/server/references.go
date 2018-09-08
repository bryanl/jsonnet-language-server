package server

import (
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
)

func textDocumentReferences(r *request, c *config.Config) (interface{}, error) {
	r.log().Info("references")
	var params lsp.ReferenceParams
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

	var lspLocations []lsp.Location
	for _, l := range locations {
		lspLocations = append(lspLocations, l.ToLSP())
	}

	return lspLocations, nil
}
