package token

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name   string
		source string
		check  func(t *testing.T, node ast.Node)
	}{
		{
			name:   "local missing body",
			source: "local a='a';",
			check: func(t *testing.T, node ast.Node) {
				local, ok := node.(*ast.Local)
				require.True(t, ok)
				body, ok := local.Body.(*partial)
				require.True(t, ok)
				expected := createLoc(1, 13)
				require.Equal(t, expected, body.Loc().Begin)
			},
		},
		{
			name:   "local bind incomplete body",
			source: "local y=o.",
			check: func(t *testing.T, node ast.Node) {
				local, ok := node.(*ast.Local)
				if assert.True(t, ok) {
					if assert.Len(t, local.Binds, 1) {
						bind := local.Binds[0]
						require.Equal(t, createIdentifier("y"), bind.Variable)
						body, ok := bind.Body.(*partial)
						if assert.True(t, ok) {
							expected := createLoc(1, 11)
							require.Equal(t, expected, body.Loc().Begin)
						}

					}

					body, ok := local.Body.(*partial)
					if assert.True(t, ok) {
						expected := createLoc(1, 11)
						require.Equal(t, expected, body.Loc().Begin)
					}
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse("file.jsonnet", tc.source)
			require.NoError(t, err)

			tc.check(t, got)
		})
	}
}

func createFakeNodeBase(l1, c1, l2, c2 int) ast.NodeBase {
	return ast.NewNodeBaseLoc(createRange("file.jsonnet", l1, c1, l2, c2))
}

func createPartial(l1, c1 int) *partial {
	return &partial{
		NodeBase: createFakeNodeBase(l1, c1, 0, 0),
	}
}

// local y=o.
