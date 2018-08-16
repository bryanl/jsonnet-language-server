package locate

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_handleImport(t *testing.T) {
	l := &Locatable{
		Loc: createRange("file.jsonnet", 1, 11, 1, 32),
	}

	i := &ast.Import{
		File: createLiteralString("import.jsonnet"),
	}

	resolved, err := l.handleImport(i)
	require.NoError(t, err)

	expected := &Resolved{
		Description: "(import) import.jsonnet",
		Location:    createRange("file.jsonnet", 1, 11, 1, 32),
	}

	assert.Equal(t, expected, resolved)
}

func createLiteralString(value string) *ast.LiteralString {
	return &ast.LiteralString{
		Kind:  ast.StringDouble,
		Value: value,
	}
}
