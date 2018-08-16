package locate

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/jlstesting"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForSpec(t *testing.T) {
	fs := ast.ForSpec{
		VarName: ast.Identifier("n"),
		Expr: &ast.Var{
			NodeBase: ast.NewNodeBaseLoc(createRange("file.jsonnet", 6, 14, 6, 19)),
			Id:       ast.Identifier("names"),
		},
	}

	source := jlstesting.Testdata(t, "for_spec1.jsonnet")

	l := &Locatable{
		Loc: createRange("file.jsonnet", 4, 1, 7, 2),
	}

	got, err := ForSpec(fs, l, source)
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 6, 5, 6, 8)

	assert.Equal(t, expected, got)
}
