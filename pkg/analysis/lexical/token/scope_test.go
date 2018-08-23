package token

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScope(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		loc      ast.Location
		expected []string
		isErr    bool
	}{
		{
			name:     "valid local",
			src:      `local a="a";a`,
			loc:      createLoc(1, 13),
			expected: []string{"a", "std"},
		},
		{
			name:     "local with no body",
			src:      `local a="a";`,
			loc:      createLoc(2, 1),
			expected: []string{"a", "std"},
		},
		{
			name:     "object keys",
			src:      `local o={a:"a"};`,
			loc:      createLoc(2, 1),
			expected: []string{"o", "std"},
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
	sm.addIdentifier(ast.Identifier("foo"))

	expectedKeys := []string{"foo"}
	require.Equal(t, expectedKeys, sm.Keys())

	expectedEntry := &ScopeEntry{
		Detail: "foo",
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
