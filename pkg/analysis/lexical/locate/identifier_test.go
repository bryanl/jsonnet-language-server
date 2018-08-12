package locate

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentifier(t *testing.T) {
	id := ast.Identifier("name")

	got, err := Identifier(id, createRange("file.jsonnet", 2, 7, 2, 20), testdata(t, "identifier1.jsonnet"))
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 2, 7, 2, 10)
	assert.Equal(t, expected, got)
}
