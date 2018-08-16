package locate

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/jlstesting"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectField(t *testing.T) {
	id := ast.Identifier("a")
	of := ast.ObjectField{
		Id:    &id,
		Expr2: &ast.Object{},
	}

	l := &Locatable{
		Token: ast.Object{},
		Loc:   createRange("file.jsonnet", 1, 11, 7, 2),
	}

	source := jlstesting.Testdata(t, "object1.jsonnet")
	got, err := ObjectField(of, l, source)
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 2, 5, 6, 6)
	assert.Equal(t, expected, got)
}
