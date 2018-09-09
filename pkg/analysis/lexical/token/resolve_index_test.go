package token

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_resolveIndex(t *testing.T) {
	testCases := []struct {
		desc        string
		indexSource string
		expected    []string
	}{
		{
			desc:        "index with var",
			indexSource: "a.b",
			expected:    []string{"a", "b"},
		},
		{
			desc:        "index with self",
			indexSource: "self.b",
			expected:    []string{"self", "b"},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			node, err := ReadSource("file.jsonnet", tC.indexSource, nil)
			require.NoError(t, err)

			spew.Dump(node)

			index, ok := node.(*ast.Index)
			require.True(t, ok)

			path := resolveIndex(index)
			assert.Equal(t, tC.expected, path)
		})
	}
}
