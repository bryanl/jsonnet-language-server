package token

import (
	"testing"

	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_scopeGraph(t *testing.T) {
	file := "file.jsonnet"

	cases := []struct {
		name   string
		source string
		pos    jpos.Position
		check  func(*testing.T, *scope)
	}{
		{
			name:   "target variable in bind",
			source: "local x=1; x",
			pos:    jpos.New(1, 7),
			check: func(t *testing.T, s *scope) {
				checkScopeIds(t, []ast.Identifier{"x"}, s)

				expectedLocations := []jpos.Location{
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 7, 1, 8)),
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 12, 1, 13)),
				}
				checkScopeRefersTo(t, expectedLocations, s, "x")
			},
		},
		{
			name:   "target var in local body",
			source: "local x=1; x",
			pos:    jpos.New(1, 12),
			check: func(t *testing.T, s *scope) {
				expectedLocations := []jpos.Location{
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 7, 1, 8)),
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 12, 1, 13)),
				}
				checkScopeRefersTo(t, expectedLocations, s, "x")
			},
		},
		{
			name:   "target parameter in bind function",
			source: "local id(x)=x; id(1)",
			pos:    jpos.New(1, 10),
			check: func(t *testing.T, s *scope) {
				expectedLocations := []jpos.Location{
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 10, 1, 11)),
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 13, 1, 14)),
				}
				checkScopeRefersTo(t, expectedLocations, s, "x")
			},
		},
		{
			name:   "target variable in function",
			source: "local id(x)=x; id(1)",
			pos:    jpos.New(1, 13),
			check: func(t *testing.T, s *scope) {
				expectedLocations := []jpos.Location{
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 10, 1, 11)),
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 13, 1, 14)),
				}
				checkScopeRefersTo(t, expectedLocations, s, "x")
			},
		},
		{
			name:   "target function in bind",
			source: "local id(x)=x; id(1)",
			pos:    jpos.New(1, 7),
			check: func(t *testing.T, s *scope) {
				expectedLocations := []jpos.Location{
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 7, 1, 9)),
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 16, 1, 18)),
				}
				checkScopeRefersTo(t, expectedLocations, s, "id")
			},
		},
		{
			name:   "target bind variable (object)",
			source: "local o={a:{b:{c:{d:'e'}}}}; o.a.b.c.d",
			pos:    jpos.New(1, 7),
			check: func(t *testing.T, s *scope) {
				expectedLocations := []jpos.Location{
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 7, 1, 8)),
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 30, 1, 31)),
				}
				checkScopeRefersTo(t, expectedLocations, s, "o")
			},
		},
		{
			name:   "target index in body",
			source: "local o={a:{b:{c:{d:'e'}}}}; o.a.b.c.d",
			pos:    jpos.New(1, 38),
			check: func(t *testing.T, s *scope) {
				expectedLocations := []jpos.Location{
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 19, 1, 20)),
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 38, 1, 39)),
				}
				checkScopeRefersTo(t, expectedLocations, s, "o", "a", "b", "c", "d")
			},
		},
		{
			name:   "target apply which points to object field",
			source: "local o={id(x)::x}; o.id(1)",
			pos:    jpos.New(1, 23),
			check: func(t *testing.T, s *scope) {
				expectedLocation := []jpos.Location{
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 10, 1, 12)),
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 23, 1, 25)),
				}
				checkScopeRefersTo(t, expectedLocation, s, "o", "id")
			},
		},
		{
			name:   "shadow: function parameter",
			source: "local x=1; local id(x)=x; id(1)",
			pos:    jpos.New(1, 21),
			check: func(t *testing.T, s *scope) {
				expectedLocation := []jpos.Location{
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 21, 1, 22)),
					jpos.NewLocation(file, jpos.NewRangeFromCoords(1, 24, 1, 25)),
				}
				checkScopeRefersTo(t, expectedLocation, s, "x")
			},
		},
	}

	for _, tc := range cases {
		if tc.name != "shadow: function parameter" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			node, err := ReadSource(file, tc.source, nil)
			require.NoError(t, err)

			nc := NewNodeCache()
			sg := scanScope(node, nc)

			_, s, err := sg.at(tc.pos)
			require.NoError(t, err)

			tc.check(t, s)
		})
	}
}

func checkScopeIds(t *testing.T, expected []ast.Identifier, s *scope) {
	got := s.ids()
	assert.Equal(t, expected, got)
}

func checkScopeRefersTo(t *testing.T, expected []jpos.Location, s *scope, id ast.Identifier, path ...string) {
	if assert.NotNil(t, s) {
		got := s.refersTo(id, path...)
		assert.Equal(t, expected, got)
	}
}
