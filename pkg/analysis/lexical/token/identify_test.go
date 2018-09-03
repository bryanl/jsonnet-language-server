package token

import (
	"testing"

	jlspos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	source1  = `local a="a"; a`
	source2  = `local o={a: "b"}; o`
	source3  = `local x=import "import.jsonnet"; x`
	source4  = `local o={a:{b:"b"}}; o.a`
	source5  = `local o={a:[1,2,3]}; o.a`
	source6  = `local o={a:{b:{c: "d"}}}; o.a.b.c.d`
	source7  = "local o={a:{b:{c: 'd'}}};\nlocal b = o.a.b;\nb.c.d"
	source8  = "local x=std.extVar('__ksonnet/params').components.x;x.item1"
	source9  = `local x=import "import.jsonnet"; local y=x.imported; y`
	source10 = `local x()=1;local y=x(); y`
	source11 = `local o={local x=1, y:x};o.y`
)

func TestIdentify(t *testing.T) {
	fieldID := createIdentifier("imported")
	importedNode := &ast.Object{
		Fields: ast.ObjectFields{
			{
				Id:    &fieldID,
				Expr2: &ast.LiteralBoolean{Value: true},
				Kind:  ast.ObjectFieldID,
			},
		},
	}

	cases := []struct {
		name     string
		source   string
		pos      jlspos.Position
		expected string
	}{
		{name: "local keyword", source: source1, pos: jlspos.New(1, 1), expected: ""},
		{name: "local bind variable", source: source1, pos: jlspos.New(1, 7), expected: `(string) "a"`},
		{name: "local body", source: source1, pos: jlspos.New(1, 14), expected: `(string) "a"`},
		{name: "object", source: source2, pos: jlspos.New(1, 7), expected: "(object) {\n  (field) a:,\n}"},
		{name: "import", source: source3, pos: jlspos.New(1, 7), expected: "(object) {\n  (field) imported::,\n}"},
		{name: "index object", source: source4, pos: jlspos.New(1, 24), expected: "(object) {\n  (field) b:,\n}"},
		{name: "index array", source: source5, pos: jlspos.New(1, 24), expected: "(array)"},
		{name: "deep nested", source: source6, pos: jlspos.New(1, 35), expected: "(string) \"d\""},
		{name: "deep nested 2", source: source7, pos: jlspos.New(3, 5), expected: "(string) \"d\""},
		{name: "local extVar assignment", source: source8, pos: jlspos.New(1, 7), expected: "(object) {\n  (field) item1:,\n}"},
		{name: "item from extVar", source: source8, pos: jlspos.New(1, 53), expected: "(object) {\n  (field) item1:,\n}"},
		{name: "nested local", source: source9, pos: jlspos.New(1, 40), expected: "(bool) true"},
		{name: "function 1", source: source10, pos: jlspos.New(1, 7), expected: "(function)"},
		{name: "function 2", source: source10, pos: jlspos.New(1, 21), expected: "(function)"},
		{name: "local 1", source: source11, pos: jlspos.New(1, 7), expected: "(object) {\n  (field) y:,\n}"},
		{name: "local 2", source: source11, pos: jlspos.New(1, 28), expected: "(number) 1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nc := NewNodeCache()
			nc.store["import.jsonnet"] = NodeEntry{Node: importedNode}

			config := IdentifyConfig{
				ExtCode: map[string]string{
					"__ksonnet/params": "{components: {x: {item1: 'param'}}}",
				},
			}

			item, err := Identify("file.jsonnet", tc.source, tc.pos, nc, config)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, item.String())
		})
	}
}
