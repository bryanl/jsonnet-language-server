package token

import (
	"testing"

	pos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDesugar(t *testing.T) {
	cases := []struct {
		name   string
		source string
		check  func(*testing.T, ast.Node)
	}{
		{
			name:   "id as object key",
			source: "local o={a:1}; o",
			check: func(t *testing.T, node ast.Node) {
				withLocal(t, node, func(local *ast.Local) {
					if assert.Len(t, local.Binds, 1) {
						o, ok := local.Binds[0].Body.(*ast.DesugaredObject)
						if assert.True(t, ok) {
							loc, ok := o.FieldLocs["a"]
							if assert.True(t, ok, "expected field a to exist") {
								assert.Equal(t, createLoc(1, 10), loc.Begin)
								assert.Equal(t, createLoc(1, 11), loc.End)
							}
						}
					}
				})
			},
		},
		{
			name:   "string as object key",
			source: "local o={'a':1}; o",
			check: func(t *testing.T, node ast.Node) {
				withLocal(t, node, func(local *ast.Local) {
					if assert.Len(t, local.Binds, 1) {
						o, ok := local.Binds[0].Body.(*ast.DesugaredObject)
						if assert.True(t, ok) {
							loc, ok := o.FieldLocs["a"]
							if assert.True(t, ok, "expected field a to exist") {
								assert.Equal(t, createLoc(1, 10), loc.Begin)
								assert.Equal(t, createLoc(1, 13), loc.End)
							}
						}
					}
				})
			},
		},
		{
			name:   "expression as object key",
			source: "local k='a', o={[k]:1}; o",
			check: func(t *testing.T, node ast.Node) {
				withLocal(t, node, func(local *ast.Local) {
					if assert.Len(t, local.Binds, 2) {
						o, ok := local.Binds[1].Body.(*ast.DesugaredObject)
						if assert.True(t, ok) {
							found := false
							for k, loc := range o.FieldLocs {
								switch k := k.(type) {
								case *ast.Var:
									if k.Id == createIdentifier("k") {
										found = true

										assert.Equal(t, createLoc(1, 18), loc.Begin)
										assert.Equal(t, createLoc(1, 19), loc.End)
									}
								}
							}

							assert.True(t, found)
						}
					}
				})
			},
		},
		{
			name:   "keep function varloc in bind",
			source: "local id(x)=x; x;",
			check: func(t *testing.T, node ast.Node) {
				withLocal(t, node, func(local *ast.Local) {
					bind := local.Binds[0]
					expectedVarLoc := pos.NewRangeFromCoords(1, 7, 1, 9)
					assert.Equal(t, expectedVarLoc.Start.ToJsonnet(), bind.VarLoc.Begin)
					assert.Equal(t, expectedVarLoc.End.ToJsonnet(), bind.VarLoc.End)
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := Parse("file.jsonnet", tc.source, nil)
			require.NoError(t, err)

			err = DesugarFile(&node)
			require.NoError(t, err)

			tc.check(t, node)
		})
	}

}
