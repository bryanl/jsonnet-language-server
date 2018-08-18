package server

import (
	"net/url"
	"os"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/pkg/errors"
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
	path, err := uriToPath(h.params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "opening file")
	}

	return lexical.HoverAtLocation(path, f, h.params.Position.Line+1, h.params.Position.Character+1)
}

func uriToPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", errors.Wrap(err, "parsing file URL")
	}

	if u.Scheme != "file" {
		return "", errors.Wrap(err, "invalid file schema")
	}

	return u.Path, nil
}
