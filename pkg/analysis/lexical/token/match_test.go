package token

import (
	"io/ioutil"
	"path/filepath"
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
		// {name: "objinside", file: "expr12.jsonnet", pos: 3, expected: 3},
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
		{name: "expr(param)", file: "expr26.jsonnet", pos: 0, expected: 3},
		{name: "id", file: "expr27.jsonnet", pos: 0, expected: 0},
		{name: "unary -", file: "expr28.jsonnet", pos: 0, expected: 1},
		{name: "unary +", file: "expr28.jsonnet", pos: 2, expected: 3},
		{name: "unary !", file: "expr28.jsonnet", pos: 4, expected: 5},
		{name: "unary !", file: "expr28.jsonnet", pos: 6, expected: 7},
		{name: "import", file: "expr29.jsonnet", pos: 0, expected: 1},
		{name: "importstr", file: "expr30.jsonnet", pos: 0, expected: 1},
		{name: "error expr", file: "expr31.jsonnet", pos: 0, expected: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

func TestMatch_Assert(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		pos      int
		expected int
		isErr    bool
	}{
		{
			name:     "assert",
			file:     "assert1.jsonnet",
			pos:      1,
			expected: 2,
		},
		{
			name:     "assert with message",
			file:     "assert2.jsonnet",
			pos:      8,
			expected: 11,
		},
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
		{
			name:     "id",
			file:     "fieldname1.jsonnet",
			pos:      4,
			expected: 4,
		},
		{
			name:     "string",
			file:     "fieldname2.jsonnet",
			pos:      4,
			expected: 4,
		},
		// {
		// 	name:     "expr",
		// 	file:     "fieldname3.jsonnet",
		// 	pos:      4,
		// 	expected: 6,
		// },
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

			printTokens(m.Tokens)

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
