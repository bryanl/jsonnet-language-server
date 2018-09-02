package langserver

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionMatchers(t *testing.T) {
	pos := position.New(2, 6)
	editRange := position.NewRange(pos, pos)

	cm := NewCompletionMatcher()

	resp := []lsp.CompletionItem{
		{
			Label: "item2",
			Kind:  lsp.CIKFile,
			TextEdit: lsp.TextEdit{
				Range:   editRange.ToLSP(),
				NewText: "item2",
			},
		},
	}

	fn := func(p position.Position, path, source string) ([]lsp.CompletionItem, error) {
		return resp, nil
	}

	err := cm.Register(`item\s?`, fn)
	require.NoError(t, err)

	list, err := cm.Match(pos, "file.jsonnet", "local item ")
	require.NoError(t, err)

	assert.Equal(t, resp, list)
}

func TestCompletionMatchers_no_match(t *testing.T) {
	pos := position.New(2, 6)
	editRange := position.NewRange(pos, pos)

	cm := NewCompletionMatcher()

	resp := []lsp.CompletionItem{
		{
			Label: "item2",
			Kind:  lsp.CIKFile,
			TextEdit: lsp.TextEdit{
				Range:   editRange.ToLSP(),
				NewText: "item2",
			},
		},
	}

	fn := func(p position.Position, path, source string) ([]lsp.CompletionItem, error) {
		return resp, nil
	}

	err := cm.Register("item", fn)
	require.NoError(t, err)

	list, err := cm.Match(pos, "file.jsonnet", "local foo ")
	require.NoError(t, err)

	expected := []lsp.CompletionItem{}
	assert.Equal(t, expected, list)
}

func TestCompletionMatchers_invalid_term(t *testing.T) {
	cm := NewCompletionMatcher()

	fn := func(p position.Position, path, source string) ([]lsp.CompletionItem, error) {
		panic("shouldn't be able to get here")
	}

	err := cm.Register("(invalid", fn)
	require.Error(t, err)
}

func TestCompletionMatchers_proceeding_ender(t *testing.T) {
	pos := position.New(2, 6)
	editRange := position.NewRange(pos, pos)

	cm := NewCompletionMatcher()

	resp := []lsp.CompletionItem{
		{
			Label: "item2",
			Kind:  lsp.CIKFile,
			TextEdit: lsp.TextEdit{
				Range:   editRange.ToLSP(),
				NewText: "item2",
			},
		},
	}

	fn := func(p position.Position, path, source string) ([]lsp.CompletionItem, error) {
		return resp, nil
	}

	err := cm.Register(`item\s?`, fn)
	require.NoError(t, err)

	list, err := cm.Match(pos, "file.jsonnet", "local item ]")
	require.NoError(t, err)

	assert.Equal(t, resp, list)
}
