package token

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_locator(t *testing.T) {
	_, err := Parse("file.jsonnet", `local a="1";`)
	node, _ := isPartialNode(err)

	n, err := locate(node, createLoc(2, 1))
	require.NoError(t, err)

	expected := &partial{
		NodeBase: ast.NewNodeBaseLoc(createRange("file.jsonnet", 1, 13, 0, 0)),
	}

	assert.Equal(t, expected, n)
}

func createRange(filename string, r1l, r1c, r2l, r2c int) ast.LocationRange {
	return ast.LocationRange{
		FileName: filename,
		Begin:    createLoc(r1l, r1c),
		End:      createLoc(r2l, r2c),
	}
}
