package token

import (
	"sort"
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type IdentifierSlice []ast.Identifier

func (p IdentifierSlice) Len() int           { return len(p) }
func (p IdentifierSlice) Less(i, j int) bool { return string(p[i]) < string(p[j]) }
func (p IdentifierSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func checkEqualEvalScope(t *testing.T, a, b *evalScope) {
	// check if stores have same keys and types
	aStoreKeys := a.keysAsID()
	sort.Sort(IdentifierSlice(aStoreKeys))
	bStoreKeys := b.keysAsID()
	sort.Sort(IdentifierSlice(bStoreKeys))

	require.Equal(t, aStoreKeys, bStoreKeys)

	for _, k := range aStoreKeys {
		require.IsType(t, a.store[k], b.store[k])
	}

	// check if references have same keys, node types, and paths
	var aRefKeys []ast.Identifier
	for k := range a.references {
		aRefKeys = append(aRefKeys, k)
	}
	sort.Sort(IdentifierSlice(aRefKeys))
	var bRefKeys []ast.Identifier
	for k := range b.references {
		bRefKeys = append(bRefKeys, k)
	}
	sort.Sort(IdentifierSlice(bRefKeys))

	require.Equal(t, aRefKeys, bRefKeys)
}

func Test_eval(t *testing.T) {
	cases := []struct {
		name          string
		node          ast.Node
		until         ast.Node
		initNodeCache func(*testing.T, *NodeCache)
		check         func(*testing.T, *evalScope, *NodeCache)
	}{
		{
			name:  "eval 1",
			node:  eval1Node,
			until: eval1Until,
			check: func(t *testing.T, got *evalScope, nc *NodeCache) {
				expected, err := newEvalScope(nc)
				require.NoError(t, err)

				expected.set("o", eval1Node.Binds[0].Body)
				checkEqualEvalScope(t, expected, got)
				// assert.Equal(t, expected.store, got.store)
				// assert.Equal(t, expected.references, got.references)

			},
		},
		{
			name:  "eval1: parent",
			node:  eval1Node,
			until: eval1Until,
			check: func(t *testing.T, got *evalScope, nc *NodeCache) {
				assert.Equal(t, eval1Node, got.parents[eval1Until])
			},
		},
		{
			name:  "eval 2: nested local",
			node:  eval2Node,
			until: eval2Until,
			check: func(t *testing.T, got *evalScope, nc *NodeCache) {
				expected, err := newEvalScope(nc)
				require.NoError(t, err)

				expected.set("o", eval2Node.Binds[0].Body)
				expected.set("b", eval2NestedLocal.Binds[0].Body)
				expected.refersTo(createIdentifier("b"), eval2Until)

				checkEqualEvalScope(t, expected, got)

				// assert.Equal(t, expected.store, got.store)
				// assert.Equal(t, expected.references, got.references)

			},
		},
		{
			name:  "eval 3: in object",
			node:  eval3Node,
			until: eval3Until,
			check: func(t *testing.T, got *evalScope, nc *NodeCache) {
				expected, err := newEvalScope(nc)
				require.NoError(t, err)

				expected.set("o", eval3Node.Binds[0].Body)
				expected.set("$", &ast.Self{})

				checkEqualEvalScope(t, expected, got)

				// assert.Equal(t, expected.store, got.store)
				// assert.Equal(t, expected.references, got.references)

			},
		},
		{
			name:  "eval 4: import",
			node:  eval4Node,
			until: eval4Until,
			initNodeCache: func(t *testing.T, nc *NodeCache) {
				ne := NodeEntry{Node: eval4ImportedNode}
				nc.store["import.jsonnet"] = ne
			},
			check: func(t *testing.T, got *evalScope, nc *NodeCache) {
				expected, err := newEvalScope(nc)
				require.NoError(t, err)

				expected.set("params", eval4ImportedNode)
				expected.refersTo(createIdentifier("params"), eval4Until)

				checkEqualEvalScope(t, expected, got)

				// assert.Equal(t, expected.store, got.store)
				// assert.Equal(t, expected.references, got.references)

			},
		},
		{
			name:  "eval 5: var references",
			node:  eval5Node,
			until: eval5Until,
			check: func(t *testing.T, got *evalScope, nc *NodeCache) {
				expected, err := newEvalScope(nc)
				require.NoError(t, err)

				expected.set("x", eval5Node.Binds[0].Body)
				expected.refersTo(createIdentifier("x"), eval5Until)

				checkEqualEvalScope(t, expected, got)

				// assert.Equal(t, expected.store, got.store)
				// assert.Equal(t, expected.references, got.references)

			},
		},
		{
			name:  "eval 6: index references",
			node:  eval6Node,
			until: eval6Until,
			check: func(t *testing.T, got *evalScope, nc *NodeCache) {
				expected, err := newEvalScope(nc)
				require.NoError(t, err)

				expected.set("x", eval6Node.Binds[0].Body)
				expected.refersTo(createIdentifier("x"), eval6Until, "a")
				expected.refersTo(createIdentifier("x"), eval6Until.Target)

				checkEqualEvalScope(t, expected, got)

				// assert.Equal(t, expected.store, got.store)
				// assert.Equal(t, expected.references, got.references)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nc := NewNodeCache()
			if tc.initNodeCache != nil {
				tc.initNodeCache(t, nc)
			}

			got, err := eval(tc.node, tc.until, nc)
			require.NoError(t, err)
			tc.check(t, got, nc)
		})
	}

}

var (
	eval1Until = &astext.Partial{}
	eval1Node  = &ast.Local{
		Binds: ast.LocalBinds{
			{
				Variable: createIdentifier("o"),
				Body: &ast.DesugaredObject{
					Fields: ast.DesugaredObjectFields{
						{
							Hide: 1,
							Name: &ast.LiteralString{Kind: 1, Value: "x"},
							Body: &ast.Local{
								Binds: ast.LocalBinds{
									{
										Variable: createIdentifier("$"),
										Body:     &ast.Self{},
									},
								},
								Body: &ast.LiteralNumber{
									Value:          1,
									OriginalString: "1",
								},
							},
						},
					},
				},
			},
		},
		Body: eval1Until,
	}

	eval2Until       = &ast.Var{Id: createIdentifier("b")}
	eval2NestedLocal = &ast.Local{
		Binds: ast.LocalBinds{
			{
				Variable: createIdentifier("b"),
				Body:     &ast.LiteralNumber{OriginalString: "2", Value: 2},
			},
		},
		Body: eval2Until,
	}
	eval2Node = &ast.Local{
		Binds: ast.LocalBinds{
			{
				Variable: createIdentifier("o"),
				Body: &ast.DesugaredObject{
					Fields: ast.DesugaredObjectFields{
						{
							Hide: 1,
							Name: &ast.LiteralString{Kind: 1, Value: "x"},
							Body: &ast.Local{
								Binds: ast.LocalBinds{
									{
										Variable: createIdentifier("$"),
										Body:     &ast.Self{},
									},
								},
								Body: &ast.LiteralNumber{
									Value:          1,
									OriginalString: "1",
								},
							},
						},
					},
				},
			},
		},
		Body: eval2NestedLocal,
	}

	eval3LocalBody = &astext.Partial{}
	eval3Until     = &astext.Partial{}
	eval3Node      = &ast.Local{
		Binds: ast.LocalBinds{
			{
				Variable: createIdentifier("o"),
				Body: &ast.DesugaredObject{
					Fields: ast.DesugaredObjectFields{
						{
							Hide: 1,
							Name: &ast.LiteralString{Kind: 1, Value: "a"},
							Body: &ast.Local{
								Binds: ast.LocalBinds{
									{
										Variable: createIdentifier("$"),
										Body:     &ast.Self{},
									},
								},
								Body: eval3Until,
							},
						},
					},
				},
			},
		},
		Body: eval3LocalBody,
	}

	eval4Until = &ast.Var{Id: createIdentifier("params")}
	eval4Node  = &ast.Local{
		Binds: ast.LocalBinds{
			{
				Variable: createIdentifier("params"),
				Body:     &ast.Import{File: createLiteralString("import.jsonnet")},
			},
		},
		Body: eval4Until,
	}
	eval4ImportedNode = createLiteralString("imported")

	eval5Until = &ast.Var{Id: createIdentifier("x")}
	eval5Node  = &ast.Local{
		Binds: ast.LocalBinds{
			{
				Variable: createIdentifier("x"),
				Body:     createLiteralString("contents"),
			},
		},
		Body: eval5Until,
	}

	eval6Var   = &ast.Var{Id: createIdentifier("x")}
	eval6Until = &ast.Index{
		Target: eval6Var,
		Index:  createLiteralString("a"),
	}
	eval6Node = &ast.Local{
		Binds: ast.LocalBinds{
			{
				Variable: createIdentifier("x"),
				Body: &ast.DesugaredObject{
					Fields: ast.DesugaredObjectFields{
						{
							Name: createLiteralString("a"),
							Body: &ast.Local{
								Binds: ast.LocalBinds{
									{
										Variable: createIdentifier("$"),
										Body:     createLiteralString("a"),
									},
								},
							},
						},
					},
				},
			},
		},
		Body: eval6Until,
	}
)
