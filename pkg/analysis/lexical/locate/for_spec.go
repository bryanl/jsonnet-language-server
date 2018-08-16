package locate

import (
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/google/go-jsonnet/ast"
)

func ForSpec(a ast.ForSpec, parent *Locatable, source string) (ast.LocationRange, error) {
	m, err := token.NewMatch(parent.Loc.FileName, source)
	if err != nil {
		return ast.LocationRange{}, err
	}

	pos, err := m.FindFirst(parent.Loc.Begin, token.TokenFor)
	if err != nil {
		return ast.LocationRange{}, err
	}

	t := m.Tokens[pos]
	r := createRange(parent.Loc.FileName,
		t.Loc.Begin.Line, t.Loc.Begin.Column,
		t.Loc.End.Line, t.Loc.End.Column)

	return r, nil
}
