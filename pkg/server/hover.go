package server

import (
	"net/url"
	"os"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical"
	"github.com/pkg/errors"
	"github.com/sourcegraph/go-langserver/pkg/lsp"
)

type hover struct {
	params lsp.TextDocumentPositionParams
}

func newHover(params lsp.TextDocumentPositionParams) *hover {
	return &hover{
		params: params,
	}
}

func (h *hover) handle() (interface{}, error) {
	u, err := url.Parse(string(h.params.TextDocument.URI))
	if err != nil {
		return nil, errors.Wrap(err, "parsing file URL")
	}

	if u.Scheme != "file" {
		return nil, errors.Wrap(err, "invalid file schema")
	}

	f, err := os.Open(u.Path)
	if err != nil {
		return nil, errors.Wrap(err, "opening file")
	}

	return lexical.HoverAtLocation(u.Path, f, h.params.Position.Line+1, h.params.Position.Character+1)
}
