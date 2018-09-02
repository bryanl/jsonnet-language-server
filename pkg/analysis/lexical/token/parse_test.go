package token

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	pos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name   string
		source string
		check  func(t *testing.T, node ast.Node)
	}{
		{
			name:   "local missing body",
			source: "local a='a';",
			check: func(t *testing.T, node ast.Node) {
				local, ok := node.(*ast.Local)
				require.True(t, ok)
				body, ok := local.Body.(*astext.Partial)
				require.True(t, ok)
				expected := createLoc(1, 13)
				require.Equal(t, expected, body.Loc().Begin)
			},
		},
		{
			name:   "local bind incomplete body",
			source: "local y=o.",
			check: func(t *testing.T, node ast.Node) {
				local, ok := node.(*ast.Local)
				if assert.True(t, ok) {
					if assert.Len(t, local.Binds, 1) {
						bind := local.Binds[0]
						require.Equal(t, createIdentifier("y"), bind.Variable)
					}

					body, ok := local.Body.(*astext.Partial)
					if assert.True(t, ok) {
						expected := createLoc(1, 11)
						require.Equal(t, expected, body.Loc().Begin)
					}
				}
			},
		},
		{
			name:   "incomplete object field",
			source: "local o={a: }; o",
			check: func(t *testing.T, node ast.Node) {
				withLocal(t, node, func(local *ast.Local) {
					if assert.Len(t, local.Binds, 1) {
						bind := local.Binds[0]
						requireIdentifier(t, "o", bind.Variable)
						o, ok := bind.Body.(*ast.Object)
						if assert.True(t, ok) {
							field := findField(t, o, "a")
							body, ok := field.Expr2.(*astext.Partial)
							if assert.True(t, ok) {
								expected := createLoc(1, 13)
								assert.Equal(t, expected, body.Loc().Begin)
							}
						}
					}
				})
			},
		},
		{
			name:   "incomplete index in local body",
			source: "local o={a: 9}; o.",
			check: func(t *testing.T, node ast.Node) {
				withLocal(t, node, func(local *ast.Local) {
					index, ok := local.Body.(*astext.PartialIndex)
					if assert.True(t, ok, "got %T; expected %T", local.Body, &astext.PartialIndex{}) {
						r := pos.NewRange(
							pos.New(1, 19),
							pos.New(1, 19))
						require.Equal(t, r, pos.FromJsonnetRange(*index.Loc()))

						v, ok := index.Target.(*ast.Var)
						if assert.True(t, ok, "got %T; expected %T", index.Target, &ast.Var{}) {
							expected := ast.Identifier("o")
							assert.Equal(t, expected, v.Id)
						}
					}
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ch := make(chan ParseDiagnostic, 1)
			done := make(chan bool, 1)

			go func() {
				for pd := range ch {
					spew.Dump(pd)
				}

				done <- true
			}()

			got, err := Parse("file.jsonnet", tc.source, ch)
			require.NoError(t, err)

			tc.check(t, got)

			<-done

		})
	}
}

func createFakeNodeBase(l1, c1, l2, c2 int) ast.NodeBase {
	return ast.NewNodeBaseLoc(createRange("file.jsonnet", l1, c1, l2, c2))
}

func createPartial(l1, c1 int) *astext.Partial {
	return &astext.Partial{
		NodeBase: createFakeNodeBase(l1, c1, 0, 0),
	}
}

type handleLocalFn func(l *ast.Local)

func withLocal(t *testing.T, node ast.Node, fn handleLocalFn) {
	local, ok := node.(*ast.Local)
	if assert.True(t, ok) {
		fn(local)
	}
}

func requireIdentifier(t *testing.T, s string, id ast.Identifier) {
	expected := createIdentifier(s)
	require.Equal(t, expected, id)
}

func findField(t *testing.T, o *ast.Object, name string) ast.ObjectField {
	for i := range o.Fields {
		field := o.Fields[i]

		if id := field.Id; id != nil {
			if string(*id) == name {
				return field
			}
		} else if field.Expr1 != nil {
			ls, ok := field.Expr1.(*ast.LiteralString)
			if ok && ls.Value == name {
				return field
			}
		}

	}

	t.Fatalf("unable to find field %s", name)
	return ast.ObjectField{}
}

func createFreeVariables(sl ...string) ast.Identifiers {
	ids := ast.Identifiers{}

	for _, s := range sl {
		ids = append(ids, createIdentifier(s))
	}

	return ids
}

func createIdentifier(s string) ast.Identifier {
	return ast.Identifier(s)
}

func idPtr(id ast.Identifier) *ast.Identifier {
	return &id
}

func createLoc(l, c int) ast.Location {
	return ast.Location{
		Line:   l,
		Column: c,
	}
}
