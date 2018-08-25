package config

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextDocument_Truncate(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		expected string
		pos      position.Position
		isErr    bool
	}{
		{
			name:     "case 1",
			source:   "123456789\n123456789",
			pos:      position.New(2, 3),
			expected: "123456789\n123",
		},
		{
			name:     "case 2",
			source:   "local foo = {\n    a: \"b\"\n};\n\nlocal y = foo.\n\nfoo\n",
			pos:      position.New(5, 15),
			expected: "local foo = {\n    a: \"b\"\n};\n\nlocal y = foo.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			td := &TextDocument{
				text: tc.source,
			}

			got, err := td.Truncate(tc.pos)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expected, got)
		})
	}

}
