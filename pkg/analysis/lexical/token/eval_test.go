package token

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/require"
)

func Test_eval(t *testing.T) {

	localBody := &astext.Partial{}

	n := &ast.Local{
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
		Body: localBody,
	}

	nc := NewNodeCache()
	got, err := eval(n, localBody, nc)
	require.NoError(t, err)

	expected := newEvalScope(nc)
	expected.set("o", n.Binds[0].Body)

	require.Equal(t, expected, got)
}

func Test_eval_nested_local(t *testing.T) {

	localBody := &ast.Var{}
	localB := &ast.Local{
		Binds: ast.LocalBinds{
			{
				Variable: createIdentifier("b"),
				Body:     &ast.LiteralNumber{OriginalString: "2", Value: 2},
			},
		},
		Body: localBody,
	}

	n := &ast.Local{
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
		Body: localB,
	}

	nc := NewNodeCache()
	got, err := eval(n, localBody, nc)
	require.NoError(t, err)

	expected := newEvalScope(nc)
	expected.set("o", n.Binds[0].Body)
	expected.set("b", localB.Binds[0].Body)

	require.Equal(t, expected, got)
}

func Test_eval_in_object(t *testing.T) {
	localBody := &astext.Partial{}
	fieldBody := &astext.Partial{}

	n := &ast.Local{
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
								Body: fieldBody,
							},
						},
					},
				},
			},
		},
		Body: localBody,
	}

	nc := NewNodeCache()
	got, err := eval(n, fieldBody, nc)
	require.NoError(t, err)

	expected := newEvalScope(nc)
	expected.set("o", n.Binds[0].Body)
	expected.set("$", &ast.Self{})

	require.Equal(t, expected, got)
}

func Test_eval_import(t *testing.T) {
	localBody := &ast.Var{}

	n := &ast.Local{
		Binds: ast.LocalBinds{
			{
				Variable: createIdentifier("params"),
				Body:     &ast.Import{File: createLiteralString("import.jsonnet")},
			},
		},
		Body: localBody,
	}

	importedNode := &ast.Var{}

	ne := NodeEntry{Node: importedNode}
	nc := NewNodeCache()
	nc.store["import.jsonnet"] = ne

	got, err := eval(n, localBody, nc)
	require.NoError(t, err)

	expected := newEvalScope(nc)
	expected.set("params", importedNode)

	require.Equal(t, expected, got)
}
