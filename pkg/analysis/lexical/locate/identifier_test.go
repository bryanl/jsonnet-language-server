package locate

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/jlstesting"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentifier_in_local_bind(t *testing.T) {
	id := ast.Identifier("name")

	l := &Locatable{
		Token: ast.LocalBind{},
		Loc:   createRange("file.jsonnet", 2, 7, 2, 20),
	}

	source := jlstesting.Testdata(t, "identifier1.jsonnet")
	got, err := Identifier(id, l, source)
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 2, 7, 2, 10)
	assert.Equal(t, expected, got)
}

func OffTestIdentifier_in_index(t *testing.T) {
	id := ast.Identifier("name")

	l := &Locatable{
		Token: ast.ObjectField{Id: &id},
		Loc:   createRange("file.jsonnet", 3, 12, 5, 10),
	}

	source := jlstesting.Testdata(t, "identifier2.jsonnet")
	got, err := Identifier(id, l, source)
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 2, 7, 2, 10)
	assert.Equal(t, expected, got)
}
