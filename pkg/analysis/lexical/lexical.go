package lexical

import (
	"io"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/google/go-jsonnet/ast"
)

var (
	emptyHover = &lsp.Hover{}
)

func HoverAtLocation(filename string, r io.Reader, l, c int, jPaths []string, cache *locate.NodeCache) (*lsp.Hover, error) {
	loc := ast.Location{
		Line:   l,
		Column: c,
	}

	v, err := newHoverVisitor(filename, r, loc)
	if err != nil {
		return nil, err
	}

	locatable, err := v.TokenAtLocation()
	if err != nil {
		return nil, err
	}

	if locatable == nil {
		return emptyHover, nil
	}

	resolved, err := locatable.Resolve(jPaths, cache)
	if err != nil {
		if err == locate.ErrUnresolvable {
			return emptyHover, nil
		}
		return nil, err
	}

	response := &lsp.Hover{
		Contents: []lsp.MarkedString{
			{
				Language: "jsonnet",
				Value:    resolved.Description,
			},
		},
	}

	if hasResolvedLocation(resolved.Location) {
		response.Range = lsp.Range{
			Start: lsp.Position{
				Line:      resolved.Location.Begin.Line - 1,
				Character: resolved.Location.Begin.Column - 1,
			},
			End: lsp.Position{
				Line:      resolved.Location.End.Line - 1,
				Character: resolved.Location.End.Column - 1,
			},
		}
	}

	return response, nil
}

func hasResolvedLocation(r ast.LocationRange) bool {
	locs := []int{r.Begin.Line, r.Begin.Column,
		r.End.Line, r.End.Column}
	for _, l := range locs {
		if l == 0 {
			return false
		}
	}
	return true
}
