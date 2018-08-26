package server

import (
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"

	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
)

type hover struct {
	params lsp.TextDocumentPositionParams
	config *config.Config
}

func newHover(params lsp.TextDocumentPositionParams, c *config.Config) *hover {
	return &hover{
		params: params,
		config: c,
	}
}

func (h *hover) handle() (interface{}, error) {
	path, err := uri.ToPath(h.params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	text, err := h.config.Text(h.params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	pos := position.FromLSPPosition(h.params.Position)

	node, err := token.Identify(path, text.String(), pos, h.config.NodeCache())
	if err != nil {
		return nil, err
	}

	response := &lsp.Hover{
		Contents: []lsp.MarkedString{
			{
				Language: "jsonnet",
				Value:    astext.TokenName(node),
			},
		},
	}

	return response, nil

	// r := strings.NewReader(text.String())

	// return lexical.HoverAtLocation(path, r, h.params.Position.Line+1, h.params.Position.Character+1, h.config)
}
