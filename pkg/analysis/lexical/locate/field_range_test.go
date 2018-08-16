package locate

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/jlstesting"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_fieldRange(t *testing.T) {
	source := jlstesting.Testdata(t, "object-source.txt")

	cases := []struct {
		name     string
		id       string
		expected ast.LocationRange
		isErr    bool
	}{
		{
			name:     "find object",
			id:       "a",
			expected: createRange("", 2, 5, 6, 6),
		},
		{
			name:     "find literal",
			id:       "b",
			expected: createRange("", 7, 5, 7, 11),
		},
		{
			name:     "find array",
			id:       "c",
			expected: createRange("", 8, 5, 8, 13),
		},
		{
			name:     "find function",
			id:       "d",
			expected: createRange("", 9, 5, 9, 14),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fieldRange(tc.id, source)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}

}
