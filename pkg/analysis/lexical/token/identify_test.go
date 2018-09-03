package token

import (
	"testing"

	jlspos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	source1  = `local a="a"; a`
	source2  = `local o={a: "b"}; o`
	source3  = `local x=import "import.jsonnet"; x.nested.x`
	source4  = `local o={a:{b:"b"}}; o.a`
	source5  = `local o={a:[1,2,3]}; o.a`
	source6  = `local o={a:{b:{c: "d"}}}; o.a.b.c`
	source7  = "local o={a:{b:{c: 'd'}}}; local b = o.a.b; b.c"
	source8  = "local x=std.extVar('__ksonnet/params').components.x;x.item1"
	source9  = `local x=import "import.jsonnet"; local y=x.imported; y`
	source10 = `local x()=1;local y=x(); y`
	source11 = `local o={local x=1, y:x};o.y`
)

func TestIdentify(t *testing.T) {
	importedSource := `{imported: true, fn(x):: [x], nested: {x: x.fn(1)}}`
	imported, err := Parse("imported.jsonnet", importedSource, nil)
	require.NoError(t, err)

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
		{name: "import 1", source: source3, pos: jlspos.New(1, 7), expected: "(object) {\n  (field) imported:,\n  (function) fn::,\n  (field) nested:,\n}"},
		{name: "import 2 ", source: source3, pos: jlspos.New(1, 43), expected: "(array)"},
		{name: "index object", source: source4, pos: jlspos.New(1, 24), expected: "(object) {\n  (field) b:,\n}"},
		{name: "index array", source: source5, pos: jlspos.New(1, 24), expected: "(array)"},
		{name: "deep nested", source: source6, pos: jlspos.New(1, 33), expected: "(string) \"d\""},
		{name: "deep nested 2", source: source7, pos: jlspos.New(1, 46), expected: "(string) 'd'"},
		{name: "extvar 1", source: source8, pos: jlspos.New(1, 7), expected: "(object) {\n  (field) item1:,\n}"},
		{name: "extvar 2", source: source8, pos: jlspos.New(1, 53), expected: "(object) {\n  (field) item1:,\n}"},
		{name: "extvar 3", source: source8, pos: jlspos.New(1, 55), expected: "(string) 'param'"},
		{name: "nested local 1", source: source9, pos: jlspos.New(1, 40), expected: "(bool) true"},
		{name: "nested local 2", source: source9, pos: jlspos.New(1, 54), expected: "(bool) true"},
		{name: "function 1", source: source10, pos: jlspos.New(1, 7), expected: "(function)"},
		{name: "function 2", source: source10, pos: jlspos.New(1, 21), expected: "(function)"},
		{name: "local 1", source: source11, pos: jlspos.New(1, 7), expected: "(object) {\n  (field) y:,\n}"},
		{name: "local 2", source: source11, pos: jlspos.New(1, 28), expected: "(number) 1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nc := NewNodeCache()
			nc.store["import.jsonnet"] = NodeEntry{Node: imported}

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
