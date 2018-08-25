package token

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_locator(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		loc      ast.Location
		expected ast.LocationRange
	}{
		{
			name:     "locate in missing object body",
			source:   `local a="1";`,
			loc:      createLoc(2, 1),
			expected: createRange("file.jsonnet", 1, 13, 0, 0),
		},
		{
			name:     "locate locate body",
			source:   `local a="1";a`,
			loc:      createLoc(1, 13),
			expected: createRange("file.jsonnet", 1, 13, 1, 14),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := Parse("file.jsonnet", tc.source)
			require.NoError(t, err)

			n, err := locate(node, tc.loc)
			require.NoError(t, err)

			require.NotNil(t, n.Loc())
			assert.Equal(t, tc.expected.String(), n.Loc().String())
		})
	}
}

func createRange(filename string, r1l, r1c, r2l, r2c int) ast.LocationRange {
	return ast.LocationRange{
		FileName: filename,
		Begin:    createLoc(r1l, r1c),
		End:      createLoc(r2l, r2c),
	}
}
