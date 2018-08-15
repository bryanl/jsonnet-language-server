package token

import (
	"io/ioutil"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatch_Bind(t *testing.T) {
	m := initmatch(t, "bind1.jsonnet")

	got, err := m.Bind(createLoc(1, 1), "x")
	require.NoError(t, err)

	expected := Tokens{
		Token{Kind: TokenIdentifier, Data: "x"},
		Token{Kind: TokenOperator, Data: "="},
		Token{Kind: TokenNumber, Data: "1"},
	}

	if assert.Equal(t, len(expected), len(got), "token count") {
		for i := range expected {
			assert.Equal(t, expected[i].Kind, got[i].Kind)
			assert.Equal(t, expected[i].Data, got[i].Data)
		}
	}
}

func TestMatch_Find(t *testing.T) {
	m := initmatch(t, "bind1.jsonnet")

	got, err := m.Find(createLoc(1, 1), TokenLocal)
	require.NoError(t, err)

	expected := 0
	require.Equal(t, expected, got)
}

func TestMatch_Expr(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		pos      int
		expected int
		isErr    bool
	}{
		{name: "null", file: "expr1.jsonnet", pos: 3, expected: 3},
		{name: "true", file: "expr2.jsonnet", pos: 3, expected: 3},
		{name: "false", file: "expr3.jsonnet", pos: 3, expected: 3},
		{name: "self", file: "expr4.jsonnet", pos: 3, expected: 3},
		{name: "$", file: "expr5.jsonnet", pos: 3, expected: 3},
		{name: "string block", file: "expr6.jsonnet", pos: 3, expected: 3},
		{name: "string double", file: "expr7.jsonnet", pos: 3, expected: 3},
		{name: "string single", file: "expr8.jsonnet", pos: 3, expected: 3},
		{name: "verbatim string double", file: "expr9.jsonnet", pos: 3, expected: 3},
		{name: "verbatim string single", file: "expr10.jsonnet", pos: 3, expected: 3},
		{name: "number", file: "expr11.jsonnet", pos: 3, expected: 3},
		{name: "objinside", file: "expr12.jsonnet", pos: 3, expected: 4},
		{name: "expr.id", file: "expr13.jsonnet", pos: 10, expected: 12},
		{name: "[] - empty array", file: "expr14.jsonnet", pos: 0, expected: 1},
		{name: "[expr]", file: "expr15.jsonnet", pos: 0, expected: 2},
		{name: "[expr,]", file: "expr16.jsonnet", pos: 0, expected: 3},
		{name: "[expr,expr]", file: "expr17.jsonnet", pos: 0, expected: 4},
		{name: "[expr,expr,]", file: "expr18.jsonnet", pos: 0, expected: 5},
		{name: "expr[expr]", file: "expr19.jsonnet", pos: 12, expected: 15},
		{name: "expr[expr:expr]", file: "expr20.jsonnet", pos: 16, expected: 21},
		{name: "expr[expr:expr:expr]", file: "expr21.jsonnet", pos: 16, expected: 23},
		{name: "expr[:expr]", file: "expr22.jsonnet", pos: 16, expected: 20},
		{name: "super.id", file: "expr23.jsonnet", pos: 0, expected: 2},
		{name: "super [expr]", file: "expr24.jsonnet", pos: 0, expected: 3},
		{name: "expr()", file: "expr25.jsonnet", pos: 0, expected: 2},
		{name: "expr(params)", file: "expr26.jsonnet", pos: 0, expected: 3},
		{name: "id", file: "expr27.jsonnet", pos: 0, expected: 0},
		{name: "unary -", file: "expr28.jsonnet", pos: 0, expected: 1},
		{name: "unary +", file: "expr40.jsonnet", pos: 0, expected: 1},
		{name: "unary !", file: "expr41.jsonnet", pos: 0, expected: 1},
		{name: "unary ~", file: "expr42.jsonnet", pos: 0, expected: 1},
		{name: "import", file: "expr29.jsonnet", pos: 0, expected: 1},
		{name: "importstr", file: "expr30.jsonnet", pos: 0, expected: 1},
		{name: "error expr", file: "expr31.jsonnet", pos: 0, expected: 1},
		{name: "assert; expr", file: "expr32.jsonnet", pos: 0, expected: 2},
		{name: "function (params) expr", file: "expr33.jsonnet", pos: 0, expected: 4},
		{name: "if/then", file: "expr34.jsonnet", pos: 0, expected: 3},
		{name: "if/then/else", file: "expr35.jsonnet", pos: 0, expected: 5},
		{name: "local bind single", file: "expr36.jsonnet", pos: 0, expected: 5},
		{name: "local bind multiple", file: "expr37.jsonnet", pos: 0, expected: 9},
		{name: "expr in super", file: "expr38.jsonnet", pos: 0, expected: 11},
		{name: "binary op", file: "expr39.jsonnet", pos: 0, expected: 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("test: %s recovered: %v: %s", tc.name, r, debug.Stack())
				}
			}()
			m := initmatch(t, tc.file)

			got, err := m.Expr(tc.pos)
			if tc.isErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestMatch_Objlocal(t *testing.T) {
	m := initmatch(t, "objlocal1.jsonnet")

	got, err := m.Objlocal(4)
	require.NoError(t, err)

	expected := 7
	assert.Equal(t, expected, got)
}

func TestMatch_ifspec(t *testing.T) {
	m := initmatch(t, "ifspec1.jsonnet")

	err := m.ifspec()
	require.NoError(t, err)

	expected := 1
	assert.Equal(t, expected, m.pos)
}

func TestMatch_forspec(t *testing.T) {
	m := initmatch(t, "forspec1.jsonnet")

	err := m.forspec()
	require.NoError(t, err)

	expected := 3
	assert.Equal(t, expected, m.pos)
}

func TestMatch_Assert(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		pos      int
		expected int
		isErr    bool
	}{
		{name: "assert", file: "assert1.jsonnet", pos: 1, expected: 2},
		{name: "assert with message", file: "assert2.jsonnet", pos: 8, expected: 11},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := initmatch(t, tc.file)

			got, err := m.Assert(tc.pos)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestMatch_Fieldname(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		pos      int
		expected int
		isErr    bool
	}{
		{name: "id", file: "fieldname1.jsonnet", pos: 4, expected: 4},
		{name: "string", file: "fieldname2.jsonnet", pos: 4, expected: 4},
		{name: "expr", file: "fieldname3.jsonnet", pos: 4, expected: 6},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := initmatch(t, tc.file)

			got, err := m.Fieldname(tc.pos)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestMatch_Objinside(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		pos      int
		expected int
		isErr    bool
	}{
		{name: "field,", file: "objinside1.jsonnet", pos: 3, expected: 8},
		{name: "field,field", file: "objinside2.jsonnet", pos: 3, expected: 11},
		{name: "local,field", file: "objinside3.jsonnet", pos: 3, expected: 16},
		{name: "[expr]: expr forspec", file: "objinside4.jsonnet", pos: 3, expected: 20},
		{name: "objlocal, [expr]: expr forspec", file: "objinside5.jsonnet", pos: 3, expected: 24},
		{name: "objlocal, [expr]: expr, objlocal, forspec", file: "objinside6.jsonnet", pos: 3, expected: 30},
		{name: "empty", file: "objinside7.jsonnet", pos: 3, expected: 4},
		// TODO: [expr]: expor forspec compspec
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := initmatch(t, tc.file)

			got, err := m.Objinside(tc.pos)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestMatch_Field(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		pos      int
		expected int
		isErr    bool
	}{
		{name: "fieldname h expr", file: "field1.jsonnet", pos: 4, expected: 6},
		{name: "fieldname + h expr", file: "field2.jsonnet", pos: 4, expected: 6},
		{name: "fieldname() h expr", file: "field3.jsonnet", pos: 4, expected: 9},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := initmatch(t, tc.file)

			got, err := m.Field(tc.pos)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestMatch_Member(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		pos      int
		expected int
		isErr    bool
	}{
		{name: "objlocal", file: "member1.jsonnet", pos: 4, expected: 7},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := initmatch(t, tc.file)

			got, err := m.Member(tc.pos)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestMatch_Params(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		pos      int
		expected int
		isErr    bool
	}{
		{name: "id", file: "params1.jsonnet", pos: 2, expected: 2},
		{name: "id,id", file: "params2.jsonnet", pos: 2, expected: 4},
		{name: "id,", file: "params3.jsonnet", pos: 2, expected: 3},
		{name: "id, id=expr", file: "params4.jsonnet", pos: 2, expected: 6},
		{name: "id, id=expr, id=expr", file: "params5.jsonnet", pos: 2, expected: 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := initmatch(t, tc.file)

			got, err := m.Params(tc.pos)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expected, got)
		})
	}
}

func createLoc(l, c int) ast.Location {
	return ast.Location{
		Line:   l,
		Column: c,
	}
}

func initmatch(t *testing.T, elem ...string) *Match {
	filename := filepath.Join(elem...)
	source := testdata(t, elem...)
	m, err := NewMatch(filename, source)
	require.NoError(t, err)
	return m
}

func testdata(t *testing.T, elem ...string) string {
	name := filepath.Join(append([]string{"testdata"}, elem...)...)
	data, err := ioutil.ReadFile(name)
	require.NoError(t, err)
	return string(data)
}
