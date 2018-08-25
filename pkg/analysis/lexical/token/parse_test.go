package token

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
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
						body, ok := bind.Body.(*astext.Partial)
						if assert.True(t, ok) {
							expected := createLoc(1, 11)
							require.Equal(t, expected, body.Loc().Begin)
						}

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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse("file.jsonnet", tc.source)
			require.NoError(t, err)

			tc.check(t, got)
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
