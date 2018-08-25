package token

import (
	"testing"

	jlspos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScope(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		loc      jlspos.Position
		expected []string
		isErr    bool
	}{
		{
			name:     "valid local",
			src:      `local a="a";a`,
			loc:      jlspos.New(1, 13),
			expected: []string{"a"},
		},
		{
			name:     "local with no body",
			src:      `local a="a";`,
			loc:      jlspos.New(2, 1),
			expected: []string{"a"},
		},
		{
			name:     "local with an incomplete body",
			src:      "local o={a:'b'};\nlocal y=o.\n",
			loc:      jlspos.New(2, 11),
			expected: []string{"o", "y"},
		},
		{
			name:     "object keys",
			src:      `local o={a:"a"};`,
			loc:      jlspos.New(2, 1),
			expected: []string{"o"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sm, err := LocationScope("file.jsonnet", tc.src, tc.loc)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, sm.Keys())
		})
	}
}

func TestScopeMap(t *testing.T) {
	sm := newScope()
	o := &ast.Object{}
	sm.addIdentifier(ast.Identifier("foo"), o)

	expectedKeys := []string{"foo"}
	require.Equal(t, expectedKeys, sm.Keys())

	expectedEntry := &ScopeEntry{
		Detail: "foo",
		Node:   o,
	}

	e, err := sm.Get("foo")
	require.NoError(t, err)

	require.Equal(t, expectedEntry, e)
}

func TestScopeMap_Get_invalid(t *testing.T) {
	sm := newScope()
	_, err := sm.Get("invalid")
	require.Error(t, err)
}

func TestScope_GetPath(t *testing.T) {
	data := &ast.DesugaredObject{
		Fields: ast.DesugaredObjectFields{
			{
				Name: createLiteralString("a"),
				Body: createLiteralString("a"),
			},
		},
	}

	o := &ast.DesugaredObject{
		Fields: ast.DesugaredObjectFields{
			{
				Name: createLiteralString("data"),
				Body: &ast.Local{
					Body: data,
				},
			},
		},
	}

	s := newScope()
	s.addIdentifier(createIdentifier("o"), o)

	cases := []struct {
		name     string
		path     []string
		expected *ScopeEntry
	}{
		{
			name: "at root",
			path: []string{"o"},
			expected: &ScopeEntry{
				Detail: "o",
				Node:   o,
			},
		},
		{
			name: "nested",
			path: []string{"o", "data"},
			expected: &ScopeEntry{
				Detail: "(object)",
				Node:   data,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.GetInPath(tc.path)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, got)
		})
	}
}

func createLiteralString(v string) *ast.LiteralString {
	return &ast.LiteralString{
		Value: v,
		Kind:  ast.StringSingle,
	}
}
