package lexical

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
)

func createFileRange(name string, r1l, r1c, r2l, r2c int) ast.LocationRange {
	return ast.LocationRange{
		FileName: name,
		Begin:    createLoc(r1l, r1c),
		End:      createLoc(r2l, r2c),
	}
}

func assertLocationRange(t *testing.T, expected, actual ast.LocationRange) {
	assert.Equal(t, expected.FileName, actual.FileName)
	assert.Equal(t, expected.Begin, actual.Begin,
		"range begin expected = %s; actual = %s",
		expected.Begin.String(), actual.Begin.String())
	assert.Equal(t, expected.End, actual.End,
		"range end expected = %s; actual = %s",
		expected.End.String(), actual.End.String())
}

func createRange(r1l, r1c, r2l, r2c int) ast.LocationRange {
	return ast.LocationRange{
		Begin: createLoc(r1l, r1c),
		End:   createLoc(r2l, r2c),
	}
}

func createLoc(line, column int) ast.Location {
	return ast.Location{Line: line, Column: column}
}
