package token

import (
	"testing"

	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHighlight(t *testing.T) {
	file := "file.jsonnet"

	cases := []struct {
		name   string
		source string
		pos    jpos.Position
		locs   []jpos.Location
		isErr  bool
	}{
		{
			name:   "bind var",
			source: "local x=1; x",
			pos:    jpos.New(1, 7),
			locs: []jpos.Location{
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 7, 1, 8)),
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 12, 1, 13)),
			},
		},
		{
			name:   "apply",
			source: "local o={id(x)::x}; o.id(1)",
			pos:    jpos.New(1, 23),
			locs: []jpos.Location{
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 10, 1, 12)),
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 23, 1, 25)),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nc := NewNodeCache()
			locations, err := Highlight(file, tc.source, tc.pos, nc)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.locs, locations)
		})
	}

}
