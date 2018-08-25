package token

import (
	"runtime/debug"
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_staticAnalyzer(t *testing.T) {
	cases := []struct {
		name          string
		source        string
		expected      []string
		scopeExpected map[string]interface{}
		loc           ast.Location
		node          func(t *testing.T, node ast.Node) ast.Node
	}{
		// {
		// 	name:     "local missing body",
		// 	source:   `local o={a:"a"};`,
		// 	expected: []string{"o", "std"},
		// 	scopeExpected: map[string]interface{}{
		// 		"o": &ast.Object{},
		// 	},
		// 	loc: createLoc(1, 17),
		// 	node: func(t *testing.T, node ast.Node) ast.Node {
		// 		local, ok := node.(*ast.Local)
		// 		require.True(t, ok)
		// 		return local.Body
		// 	},
		// },
		// {
		// 	name:     "inside a function body",
		// 	source:   `local fn(x) = x; fn("1")`,
		// 	expected: []string{"fn", "std"},
		// 	scopeExpected: map[string]interface{}{
		// 		"fn": &ast.Apply{},
		// 	},
		// 	loc: createLoc(1, 17),
		// 	node: func(t *testing.T, node ast.Node) ast.Node {
		// 		local, ok := node.(*ast.Local)
		// 		require.True(t, ok)

		// 		return local.Body
		// 	},
		// },
		// {
		// 	name:     "inside apply function body",
		// 	source:   `local fn(x) = x+1; fn(1)`,
		// 	expected: []string{"fn", "std", "x"},
		// 	loc:      createLoc(1, 15),
		// 	scopeExpected: map[string]interface{}{
		// 		"fn": &ast.Binary{},
		// 		"x":  &ast.Binary{},
		// 	},
		// 	node: func(t *testing.T, node ast.Node) ast.Node {
		// 		local, ok := node.(*ast.Local)
		// 		require.True(t, ok)
		// 		require.Len(t, local.Binds, 1)
		// 		bind := local.Binds[0]
		// 		require.NotNil(t, bind.Fun)
		// 		require.NotNil(t, bind.Fun.Body)
		// 		return bind.Fun.Body
		// 	},
		// },
		// {
		// 	name:     "function field member in object",
		// 	source:   `local o = {fn(x):x}; o.fn(1)`,
		// 	expected: []string{"fn", "o", "std", "x"},
		// 	loc:      createLoc(1, 18),
		// 	scopeExpected: map[string]interface{}{
		// 		"fn": &ast.Var{},
		// 		"o":  &ast.Object{},
		// 	},
		// 	node: func(t *testing.T, node ast.Node) ast.Node {
		// 		local, ok := node.(*ast.Local)
		// 		require.True(t, ok)
		// 		require.Len(t, local.Binds, 1)
		// 		bind := local.Binds[0]
		// 		o, ok := bind.Body.(*ast.Object)
		// 		require.True(t, ok)
		// 		require.Len(t, o.Fields, 1)
		// 		field := o.Fields[0]
		// 		spew.Dump(field.Expr2)
		// 		return field.Expr2
		// 	},
		// },
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Recovered in f: %s\n%v", string(debug.Stack()), r)
				}
			}()

			node, err := Parse("file.jsonnet", tc.source)
			require.NoError(t, err)

			sc, err := analyze(node, tc.loc)
			require.NoError(t, err)

			expected := createFreeVariables(tc.expected...)

			freeVars := tc.node(t, node).FreeVariables()
			if assert.Equal(t, expected, freeVars) {
				for _, v := range freeVars {
					id := string(v)
					if id == "std" {
						// not handling std yet
						continue
					}

					node, ok := sc.store[id]
					if assert.True(t, ok, "unable to find free variable %s", id) {
						require.IsType(t, tc.scopeExpected[id], node,
							"expected scope item %q to be a %T; it was a %T", id, tc.scopeExpected[id], node)
					}
				}
			}
		})
	}
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
