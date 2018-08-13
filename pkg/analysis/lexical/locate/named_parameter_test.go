package locate

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamedParameter(t *testing.T) {
	p := ast.NamedParameter{
		Name: ast.Identifier("x"),
		DefaultArg: &ast.LiteralNumber{
			Value:          1,
			OriginalString: "1",
		},
	}

	source := testdata(t, "named_parameter1.jsonnet")
	got, err := NamedParameter(p, createRange("file.jsonnet", 1, 7, 1, 18), source)
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 1, 9, 1, 12)

	assert.Equal(t, expected, got)
}
