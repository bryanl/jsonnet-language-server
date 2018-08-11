package server

import (
	"fmt"
	"net/url"
	"os"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical"
	"github.com/google/go-jsonnet/ast"
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

	loc := ast.Location{
		Line:   h.params.Position.Line + 1,
		Column: h.params.Position.Character,
	}

	locatable, err := lexical.TokenAtLocation(u.Path, f, loc)
	if err != nil {
		return nil, err
	}

	response := &lsp.Hover{
		Contents: []lsp.MarkedString{
			{
				Language: "markdown",
				Value:    fmt.Sprintf("%T", locatable.Token),
			},
		},
		Range: lsp.Range{
			Start: lsp.Position{Line: locatable.Loc.Begin.Line - 1, Character: locatable.Loc.Begin.Column - 1},
			End:   lsp.Position{Line: locatable.Loc.End.Line - 1, Character: locatable.Loc.End.Column - 1},
		},
	}

	return response, nil
}
