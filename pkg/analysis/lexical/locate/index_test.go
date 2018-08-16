package locate

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/jlstesting"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndex(t *testing.T) {
	idxID := ast.Identifier("nested2")

	cases := []struct {
		name         string
		file         string
		idx          *ast.Index
		expected     ast.LocationRange
		notLocatable bool
		isErr        bool
	}{
		{
			name: "with id",
			file: "index1.jsonnet",
			idx: &ast.Index{
				Id:       &idxID,
				NodeBase: ast.NewNodeBaseLoc(createRange("file.jsonnet", 11, 13, 11, 22)),
			},
			expected: createRange("file.jsonnet", 11, 15, 11, 21),
		},
		{
			name: "with index",
			file: "index2.jsonnet",
			idx: &ast.Index{
				Target: &ast.Var{
					Id: ast.Identifier("a"),
				},
				Index: &ast.LiteralNumber{Value: 2, OriginalString: "2"},
			},
			notLocatable: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := &Locatable{}

			source := jlstesting.Testdata(t, tc.file)

			got, err := Index(tc.idx, l, source)
			if tc.isErr && !tc.notLocatable {
				require.Error(t, err)
				return
			} else if tc.notLocatable {
				require.Error(t, ErrNotLocatable, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)

		})
	}
}
