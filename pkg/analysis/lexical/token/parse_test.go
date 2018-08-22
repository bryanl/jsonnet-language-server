package token

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	_, err := Parse("file.jsonnet", `local a="a";`)
	require.Error(t, err)

	node, isPartial := isPartialNode(err)
	assert.IsType(t, &ast.Local{}, node)
	assert.True(t, isPartial)
}
