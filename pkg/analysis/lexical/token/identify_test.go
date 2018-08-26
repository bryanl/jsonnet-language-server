package token

import (
	"testing"

	jlspos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentify(t *testing.T) {
	fieldID := createIdentifier("imported")
	importedNode := &ast.Object{
		Fields: ast.ObjectFields{
			{
				Id:    &fieldID,
				Expr2: &ast.LiteralBoolean{Value: true},
			},
		},
	}

	source1 := `local a="a"; a`
	source2 := `local o={a: "b"}; o`
	source3 := `local x=import "import.jsonnet"; x`
	source4 := `local o={a:{b:"b"}}; o.a`
	source5 := `local o={a:[1,2,3]}; o.a`
	source6 := `local o={a:{b:{c: "d"}}}; o.a.b.c.d`
	source7 := "local o={a:{b:{c: 'd'}}};\nlocal b = o.a.b;\nb.c.d"

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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nc := NewNodeCache()
			nc.store["import.jsonnet"] = NodeEntry{Node: importedNode}

			item, err := Identify("file.jsonnet", tc.source, tc.pos, nc)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, item.String())
		})
	}
}
