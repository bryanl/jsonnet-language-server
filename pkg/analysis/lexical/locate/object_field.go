package locate

import (
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/google/go-jsonnet/ast"
	"github.com/sirupsen/logrus"
)

// ObjectField locates object fields.
func ObjectField(field ast.ObjectField, parentRange ast.LocationRange, source string) (ast.LocationRange, error) {
	parentSource, err := extractRange(source, parentRange)
	if err != nil {
		return ast.LocationRange{}, err
	}

	fieldName := astext.ObjectFieldName(field)
	m, err := token.NewMatch("", source)
	if err != nil {
		return ast.LocationRange{}, err
	}

	logrus.Printf("looking for object field %s in %s", fieldName, parentSource)
	tokens, err := m.FindObjectField(parentRange.Begin, fieldName)
	if err != nil {
		return ast.LocationRange{}, err
	}

	begin := tokens[0].Loc.Begin
	end := tokens[len(tokens)-1].Loc.End
	r := createRange("", begin.Line, begin.Column, end.Line, end.Column)
	return r, nil
}
