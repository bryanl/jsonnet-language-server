package server

import (
	"net/url"
	"os"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

type hover struct {
	params lsp.TextDocumentPositionParams
	config *Config
}

func newHover(params lsp.TextDocumentPositionParams, c *Config) *hover {
	return &hover{
		params: params,
		config: c,
	}
}

func (h *hover) handle() (interface{}, error) {
	path, err := uriToPath(h.params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	/* nosec */
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "opening file")
	}

	return lexical.HoverAtLocation(path, f, h.params.Position.Line+1, h.params.Position.Character+1, h.config.JsonnetLibPaths(), h.config.NodeCache())
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

func posToLoc(pos lsp.Position) ast.Location {
	return ast.Location{
		Line:   pos.Line + 1,
		Column: pos.Character + 1,
	}
}
