package token

import (
	"testing"

	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func offTest_scopeGraph(t *testing.T) {
	file := "file.jsonnet"

	cases := []struct {
		name   string
		source string
		check  func(*testing.T, *scopeGraph)
	}{
		{
			name:   "target variable in bind",
			source: "local x=1; x",
			check: func(t *testing.T, sg *scopeGraph) {
				_, s, err := sg.at(jpos.New(1, 12))
				require.NoError(t, err)

				decls := s.declarations()
				assert.True(t, decls.contains(ast.Identifier("x")))
			},
		},
		{
			name:   "target parameter in bind function",
			source: "local id(x)=x; id(1)",
			check: func(t *testing.T, sg *scopeGraph) {
			},
		},
		{
			name:   "target variable in function",
			source: "local id(x)=x; id(1)",
			check: func(t *testing.T, sg *scopeGraph) {
			},
		},
		{
			name:   "target function in bind",
			source: "local id(x)=x; id(1)",
			check: func(t *testing.T, sg *scopeGraph) {
			},
		},
		{
			name:   "target bind variable (object)",
			source: "local o={a:{b:{c:{d:'e'}}}}; o.a.b.c.d",
			check: func(t *testing.T, sg *scopeGraph) {
			},
		},
		{
			name:   "target index in body",
			source: "local o={a:{b:{c:{d:'e'}}}}; o.a.b.c.d",
			check: func(t *testing.T, sg *scopeGraph) {
			},
		},
		{
			name:   "target apply which points to object field",
			source: "local o={id(x)::x}; o.id(1)",
			check: func(t *testing.T, sg *scopeGraph) {
			},
		},
		{
			name:   "shadow: function parameter",
			source: "local x=1; local id(x)=x; id(1)",
			check: func(t *testing.T, sg *scopeGraph) {
			},
		},
		{
			name:   "target in array",
			source: "local x=1, i=1; local a=[x]; a[i]",
			check: func(t *testing.T, sg *scopeGraph) {
			},
		},
		{
			name:   "self",
			source: `{person1: {name: "Alice", welcome: "Hello " + self.name + "!",}, person2: self.person1 {name: "Bob"}}`,
			check: func(t *testing.T, sg *scopeGraph) {
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := ReadSource(file, tc.source, nil)
			require.NoError(t, err)

			nc := NewNodeCache()
			sg := scanScope(node, nc)

			tc.check(t, sg)
		})
	}
}
