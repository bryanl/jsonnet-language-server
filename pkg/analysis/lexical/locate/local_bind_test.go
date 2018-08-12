package locate

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalBind(t *testing.T) {
	bind := ast.LocalBind{
		Variable: ast.Identifier("name"),
		Body: &ast.LiteralString{
			NodeBase: ast.NewNodeBase(createRange("file.jsonnet", 2, 14, 2, 20), nil),
			Value:    "name",
			Kind:     1,
		},
	}

	got, err := LocalBind(bind, createRange("file.jsonnet", 2, 1, 4, 3), testdata(t, "local_bind1.jsonnet"))
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 2, 7, 2, 20)
	assert.Equal(t, expected, got)
}

func TestLocalBind_function(t *testing.T) {
	bind := ast.LocalBind{
		Variable: ast.Identifier("fn"),
		Body: &ast.Function{
			NodeBase: ast.NewNodeBaseLoc(createRange("file.jsonnet", 1, 7, 1, 16)),
			Parameters: ast.Parameters{
				Required: ast.Identifiers{"x"},
			},
			Body: &ast.LiteralNumber{
				NodeBase:       ast.NewNodeBaseLoc(createRange("file.jsonnet", 1, 15, 1, 16)),
				Value:          9,
				OriginalString: "9",
			},
		},
	}

	source := testdata(t, "local_bind2.jsonnet")
	got, err := LocalBind(bind, createRange("file.jsonnet", 1, 1, 3, 3), source)
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 1, 7, 1, 16)
	assert.Equal(t, expected, got)
}

func testdata(t *testing.T, elem ...string) string {
	name := filepath.Join(append([]string{"testdata"}, elem...)...)
	data, err := ioutil.ReadFile(name)
	require.NoError(t, err)
	return string(data)
}
