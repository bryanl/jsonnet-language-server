package lexical

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/jlstesting"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_fieldRange(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		field    string
		expected ast.LocationRange
		isErr    bool
	}{
		{
			name:     "double quoted value",
			source:   "object1.jsonnet",
			field:    "a",
			expected: createRange(2, 5, 2, 11),
		},
		{
			name:     "single quoted value",
			source:   "object2.jsonnet",
			field:    "a",
			expected: createRange(2, 5, 2, 11),
		},
		{
			name:     "block quoted value",
			source:   "object3.jsonnet",
			field:    "a",
			expected: createRange(2, 5, 5, 8),
		},
		{
			name:     "double quoted value with escaped double quotes",
			source:   "object4.jsonnet",
			field:    "a",
			expected: createRange(2, 5, 2, 15),
		},
		{
			name:     "array value",
			source:   "object5.jsonnet",
			field:    "a",
			expected: createRange(2, 5, 2, 15),
		},
		{
			name:     "object value",
			source:   "object6.jsonnet",
			field:    "a",
			expected: createRange(2, 5, 2, 14),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := jlstesting.Testdata(t, "fieldRange", tc.source)

			got, err := fieldRange(tc.field, source)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expected, got)

		})
	}
}
