package locate

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredParameter(t *testing.T) {
	p := astext.RequiredParameter{
		ID: ast.Identifier("x"),
	}

	source := testdata(t, "required_parameter1.jsonnet")
	got, err := RequiredParameter(p, createRange("file.jsonnet", 1, 7, 1, 16), source)
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 1, 10, 1, 10)

	assert.Equal(t, expected, got)
}

func TestRequiredParameter_subsequent(t *testing.T) {
	p := astext.RequiredParameter{
		ID: ast.Identifier("y"),
	}

	source := testdata(t, "required_parameter2.jsonnet")
	got, err := RequiredParameter(p, createRange("file.jsonnet", 1, 7, 1, 16), source)
	require.NoError(t, err)

	expected := createRange("file.jsonnet", 1, 12, 1, 12)

	assert.Equal(t, expected, got)
}
