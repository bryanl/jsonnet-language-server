package locate

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

func DesugaredObjectField(field ast.DesugaredObjectField, parentRange ast.LocationRange, source string) (ast.LocationRange, error) {
	parentSource, err := extractRange(source, parentRange)
	if err != nil {
		return ast.LocationRange{}, err
	}

	// TODO get value from a node
	fieldName := ""
	switch t := field.Name.(type) {
	case *ast.LiteralString:
		fieldName = t.Value
	default:
		return ast.LocationRange{}, errors.Errorf("unable to get desugared field name from type %T", t)
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

func extractRange(source string, r ast.LocationRange) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(source))
	scanner.Split(bufio.ScanRunes)

	var buf bytes.Buffer

	col := 1
	line := 1

	for scanner.Scan() {
		cur := scanner.Text()
		if cur == "\n" {
			line++
			col = 1
		}

		loc := ast.Location{Line: line, Column: col}
		if inRange(loc, r) {
			if _, err := buf.WriteString(cur); err != nil {
				return "", err
			}
		}

		col++
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func inRange(l ast.Location, r ast.LocationRange) bool {
	if r.Begin.Line == l.Line {
		return r.Begin.Column <= l.Column
	} else if r.Begin.Line <= l.Line && r.End.Line >= l.Line {
		return true
	}

	return false
}
