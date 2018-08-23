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
		name     string
		source   string
		expected []string
		node     func(t *testing.T, node ast.Node) ast.Node
	}{
		{
			name:     "normal case",
			source:   `local o={a:"a"}; o`,
			expected: []string{"o", "std"},
			node: func(t *testing.T, node ast.Node) ast.Node {
				local, ok := node.(*ast.Local)
				require.True(t, ok)
				return local.Body
			},
		},
		{
			name:     "local missing body",
			source:   `local o={a:"a"};`,
			expected: []string{"o", "std"},
			node: func(t *testing.T, node ast.Node) ast.Node {
				local, ok := node.(*ast.Local)
				require.True(t, ok)
				return local.Body
			},
		},
		{
			name:     "inside a function body",
			source:   `local fn(x) = x; fn("1")`,
			expected: []string{"fn", "std"},
			node: func(t *testing.T, node ast.Node) ast.Node {
				local, ok := node.(*ast.Local)
				require.True(t, ok)

				return local.Body
			},
		},
		{
			name:     "inside apply function body",
			source:   `local fn(x) = x; fn("1")`,
			expected: []string{"fn", "std", "x"},
			node: func(t *testing.T, node ast.Node) ast.Node {
				local, ok := node.(*ast.Local)
				require.True(t, ok)
				require.Len(t, local.Binds, 1)
				bind := local.Binds[0]
				require.NotNil(t, bind.Fun)
				require.NotNil(t, bind.Fun.Body)
				return bind.Fun.Body
			},
		},
		{
			name:     "function field member in object",
			source:   `local o = {fn(x):x}; o.fn(1)`,
			expected: []string{"o", "std", "x"},
			node: func(t *testing.T, node ast.Node) ast.Node {
				local, ok := node.(*ast.Local)
				require.True(t, ok)
				require.Len(t, local.Binds, 1)
				bind := local.Binds[0]
				o, ok := bind.Body.(*ast.Object)
				require.True(t, ok)
				require.Len(t, o.Fields, 1)
				field := o.Fields[0]
				return field.Expr2
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Recovered in f: %s\n%v", string(debug.Stack()), r)
				}
			}()

			node, err := Parse("file.jsonnet", tc.source)
			if err != nil {
				node, _ = isPartialNode(err)
			}

			err = analyze(node)
			require.NoError(t, err)

			expected := createFreeVariables(tc.expected...)

			assert.Equal(t, expected, tc.node(t, node).FreeVariables())
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
