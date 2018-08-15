package locate

import (
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/google/go-jsonnet/ast"
)

// ObjectField locates object fields.
func ObjectField(field ast.ObjectField, parentRange ast.LocationRange, source string) (ast.LocationRange, error) {
	fieldName := astext.ObjectFieldName(field)
	m, err := token.NewMatch("", source)
	if err != nil {
		return ast.LocationRange{}, err
	}

	tokens, err := m.FindObjectField(parentRange.Begin, fieldName)
	if err != nil {
		return ast.LocationRange{}, err
	}

	begin := tokens[0].Loc.Begin
	end := tokens[len(tokens)-1].Loc.End
	r := createRange("", begin.Line, begin.Column, end.Line, end.Column)
	return r, nil
}
