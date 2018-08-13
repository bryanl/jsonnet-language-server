package locate

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func OffTestForSpec(t *testing.T) {
	fs := ast.ForSpec{
		VarName: ast.Identifier("n"),
		Expr: &ast.Var{
			NodeBase: ast.NewNodeBaseLoc(createRange("file.jsonnet", 6, 14, 6, 19)),
			Id:       ast.Identifier("names"),
		},
	}

	source := testdata(t, "for_spec1.jsonnet")
	got, err := ForSpec(fs, createRange("file.jsonnet", 4, 1, 7, 2), source)
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 6, 5, 6, 18)

	assert.Equal(t, expected, got)
}
