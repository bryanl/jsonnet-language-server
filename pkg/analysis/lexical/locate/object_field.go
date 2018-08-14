package locate

import (
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

// ObjectField locates object fields.
func ObjectField(field ast.ObjectField, parentRange ast.LocationRange, source string) (ast.LocationRange, error) {
	parentSource, err := extractRange(source, parentRange)
	if err != nil {
		return ast.LocationRange{}, err
	}

	// TODO get value from a node
	fieldName := ""

	if field.Id != nil {
		fieldName = string(*field.Id)
	} else if field.Expr1 != nil {
		fieldName = astext.TokenValue(field.Expr1)
	} else {
		return ast.LocationRange{}, errors.New("field doesn't have a name")
	}

	if strings.Contains(fieldName, "unknown") {
		return ast.LocationRange{}, errors.Errorf("can't parse object field name %q", fieldName)
	}

	fieldLocation, err := fieldRange(fieldName, parentSource)
	if err != nil {
		return ast.LocationRange{}, err
	}

	fieldLocation.FileName = parentRange.FileName
	fieldLocation.Begin.Line += parentRange.Begin.Line - 1
	fieldLocation.End.Line += parentRange.Begin.Line - 1

	return fieldLocation, nil
}
