package server

import (
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"

	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
)

var (
	emptyHover = &lsp.Hover{}
)

type hover struct {
	params lsp.TextDocumentPositionParams
	config *config.Config
	path   string
}

func newHover(params lsp.TextDocumentPositionParams, c *config.Config) (*hover, error) {
	path, err := uri.ToPath(params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	return &hover{
		params: params,
		config: c,
		path:   path,
	}, nil
}

func (h *hover) handle() (interface{}, error) {
	text, err := h.config.Text(h.params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	pos := position.FromLSPPosition(h.params.Position)

	ic, err := token.NewIdentifyConfig(h.path, h.config.JsonnetLibPaths()...)
	if err != nil {
		return nil, err
	}

	item, err := token.Identify(text.String(), pos, h.config.NodeCache(), ic)
	if err != nil {
		return nil, err
	}

	value := item.String()
	if value == "" {
		return emptyHover, nil
	}

	response := &lsp.Hover{
		Contents: []lsp.MarkedString{
			{
				Language: "jsonnet",
				Value:    value,
			},
		},
	}

	return response, nil
}
