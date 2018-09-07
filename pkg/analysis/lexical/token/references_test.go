package token

import (
	"testing"

	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReferences(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		pos      jpos.Position
		expected []jpos.Location
	}{
		{
			name:   "target bind variable literal",
			source: "local x=1; x",
			pos:    jpos.New(1, 7),
			expected: []jpos.Location{
				jpos.NewLocation("file.jsonnet", jpos.NewRangeFromCoords(1, 7, 1, 10)),
				jpos.NewLocation("file.jsonnet", jpos.NewRangeFromCoords(1, 12, 1, 13)),
			},
		},
		{
			name:   "target bind variable with object",
			source: "local x={a:'a'}; x.a",
			pos:    jpos.New(1, 7),
			expected: []jpos.Location{
				jpos.NewLocation("file.jsonnet", jpos.NewRangeFromCoords(1, 7, 1, 16)),
				jpos.NewLocation("file.jsonnet", jpos.NewRangeFromCoords(1, 18, 1, 19)),
			},
		},
		{
			name:   "target key inside bind variable with object",
			source: "local x={a:'a'}; x.a",
			pos:    jpos.New(1, 10),
			expected: []jpos.Location{
				jpos.NewLocation("file.jsonnet", jpos.NewRangeFromCoords(1, 10, 1, 11)),
				jpos.NewLocation("file.jsonnet", jpos.NewRangeFromCoords(1, 18, 1, 21)),
			},
		},
		{
			name:   "target key nested inside bind variable with object",
			source: "local x={a:{b:'b'}}; x.a.b",
			pos:    jpos.New(1, 13),
			expected: []jpos.Location{
				jpos.NewLocation("file.jsonnet", jpos.NewRangeFromCoords(1, 13, 1, 14)),
				jpos.NewLocation("file.jsonnet", jpos.NewRangeFromCoords(1, 22, 1, 27)),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nc := NewNodeCache()

			locations, err := References("file.jsonnet", tc.source, tc.pos, nc)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, locations)
		})
	}

}
