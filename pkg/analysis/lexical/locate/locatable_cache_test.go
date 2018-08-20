package locate

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocatableCache_GetAtPosition(t *testing.T) {
	lc := NewLocatableCache()

	list := []Locatable{
		{Loc: createRange("r1", 1, 1, 10, 10)},
		{Loc: createRange("r2", 3, 1, 10, 10)},
		{Loc: createRange("r3", 5, 1, 9, 7)},
	}

	err := lc.Store("a", list)
	require.NoError(t, err)

	cases := []struct {
		name     string
		pos      ast.Location
		expected string
	}{
		{
			name:     "r2",
			pos:      createLoc(4, 7),
			expected: "r2",
		},
		{
			name:     "r3",
			pos:      createLoc(6, 3),
			expected: "r3",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := lc.GetAtPosition("a", tc.pos)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, got.Loc.FileName)
		})
	}

}
