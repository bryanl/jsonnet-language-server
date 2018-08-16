package locate

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/jlstesting"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func OffTestDesugaredObjectField(t *testing.T) {
	field := ast.DesugaredObjectField{
		Hide: 1,
		Name: &ast.LiteralString{
			Value: "a",
			Kind:  1,
		},
		Body: &ast.Local{
			NodeBase: ast.NewNodeBaseLoc(createRange("file.jsonnet", 2, 8, 2, 11)),
			Binds: ast.LocalBinds{
				{
					Variable: ast.Identifier("$"),
					Body:     &ast.Self{},
				},
			},
			Body: &ast.LiteralString{
				NodeBase: ast.NewNodeBaseLoc(createRange("file.jsonnet", 2, 8, 2, 11)),
				Value:    "a",
				Kind:     1,
			},
		},
	}

	source := jlstesting.Testdata(t, "desugared_object1.jsonnet")
	got, err := DesugaredObjectField(field, createRange("file.jsonnet", 1, 13, 3, 2), source)
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 2, 5, 2, 11)
	assert.Equal(t, expected, got)
}
