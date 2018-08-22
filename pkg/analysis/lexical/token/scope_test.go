package token

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScope(t *testing.T) {
	sm, err := LocationScope("file.jsonnet", `local a="a";`, createLoc(2, 1))
	require.NoError(t, err)

	expected := []string{"a", "std"}

	assert.Equal(t, expected, sm.Keys())
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
