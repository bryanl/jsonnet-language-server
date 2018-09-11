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
		name      string
		source    string
		positions []jpos.Position
		locs      []jpos.Location
		isErr     bool
	}{
		{
			name:      "bind var",
			source:    "local x=1; x",
			positions: []jpos.Position{jpos.New(1, 7), jpos.New(1, 12)},
			locs: []jpos.Location{
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 7, 1, 8)),
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 12, 1, 13)),
			},
		},
		{
			name:      "target parameter in bind function",
			source:    "local id(x)=x; id(1)",
			positions: []jpos.Position{jpos.New(1, 10), jpos.New(1, 13)},
			locs: []jpos.Location{
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 10, 1, 11)),
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 13, 1, 14)),
			},
		},
		// {
		// 	name:      "apply",
		// 	source:    "local o={id(x)::x}; o.id(1)",
		// 	positions: []jpos.Position{jpos.New(1, 10), jpos.New(1, 23)},
		// 	locs: []jpos.Location{
		// 		jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 10, 1, 12)),
		// 		jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 23, 1, 25)),
		// 	},
		// },
		{
			name:      "shadow: function parameter",
			source:    "local x=1; local id(x)=x; id(1)",
			positions: []jpos.Position{jpos.New(1, 21), jpos.New(1, 24)},
			locs: []jpos.Location{
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 21, 1, 22)),
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 24, 1, 25)),
			},
		},
		{
			name:      "target in array index",
			source:    "local x=1, i=1; local a=[x]; a[i]",
			positions: []jpos.Position{jpos.New(1, 32)},
			locs: []jpos.Location{
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 12, 1, 13)),
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 32, 1, 33)),
			},
		},
		{
			name:      "target index in body",
			source:    "local o={a:{b:{c:{d:'e'}}}}; o.a.b.c.d",
			positions: []jpos.Position{jpos.New(1, 38), jpos.New(1, 19)},
			locs: []jpos.Location{
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 19, 1, 20)),
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 38, 1, 39)),
			},
		},

		{
			name:      "self",
			source:    `{n: 1, m: self.n + 1}`,
			positions: []jpos.Position{jpos.New(1, 16)},
			locs: []jpos.Location{
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 16, 1, 17)),
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 2, 1, 3)),
			},
		},
		{
			name:      "self nested",
			source:    `{person1: {name: "Alice", welcome: "Hello " + self.name + "!"}, person2: self.person1 {name: "Bob"}}`,
			positions: []jpos.Position{jpos.New(1, 52)},
			locs: []jpos.Location{
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 12, 1, 16)),
				jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 52, 1, 56)),
			},
		},
	}

	for _, tc := range cases {
		if tc.name != "target index in body" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			for _, pos := range tc.positions {
				nc := NewNodeCache()
				locations, err := Highlight(file, tc.source, pos, nc)
				if tc.isErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)

				expected := jpos.Locations{}
				for _, l := range tc.locs {
					expected.Add(l)
				}

				assert.True(t, expected.Equal(locations), "\n\texpected %s\n\tgot: %s",
					expected.String(), locations.String())
			}

		})
	}

}
