package token

import (
	"testing"

	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_pathToLocation(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		pos      jpos.Position
		expected objectPath
		isErr    bool
	}{
		{
			name:   "position in field name",
			source: "{a:'a'}",
			pos:    jpos.New(1, 2),
			expected: objectPath{
				path: []string{"a"},
				loc:  jpos.NewRangeFromCoords(1, 2, 1, 3),
			},
		},
		{
			name:   "position in field body",
			source: "{a:'a'}",
			pos:    jpos.New(1, 5),
			expected: objectPath{
				path: []string{"a"},
				loc:  jpos.NewRangeFromCoords(1, 2, 1, 3),
			},
		},
		{
			name:   "position in field body and body is object",
			source: "{a:{b:'b'}}",
			pos:    jpos.New(1, 5),
			expected: objectPath{
				path: []string{"a", "b"},
				loc:  jpos.NewRangeFromCoords(1, 5, 1, 6),
			},
		},
		{
			name:   "position in field with string name",
			source: "{'a': 'a'}",
			pos:    jpos.New(1, 3),
			expected: objectPath{
				path: []string{"a"},
				loc:  jpos.NewRangeFromCoords(1, 2, 1, 5),
			},
		},
		{
			name:   "position in field with expression name",
			source: "{[a]: 'a'}",
			pos:    jpos.New(1, 3),
			isErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := ReadSource("file.jsonnet", tc.source, nil)
			require.NoError(t, err)

			withDesugaredObject(t, node, func(o *ast.DesugaredObject) {
				path, err := pathToLocation(o, tc.pos)
				if tc.isErr {
					require.Error(t, err)
					return
				}

				require.NoError(t, err)
				assert.Equal(t, tc.expected, path)
			})
		})
	}
}

func withDesugaredObject(t *testing.T, n ast.Node, fn func(o *ast.DesugaredObject)) {
	o, ok := n.(*ast.DesugaredObject)
	require.True(t, ok)

	fn(o)
}
