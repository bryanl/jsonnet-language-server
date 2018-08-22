package token

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScope(t *testing.T) {
	ids, err := Scope("file.jsonnet", `local a="a";`, createLoc(2, 1))
	require.NoError(t, err)

	expected := ast.Identifiers{
		ast.Identifier("a"),
		ast.Identifier("std"),
	}

	assert.Equal(t, expected, ids)
}
