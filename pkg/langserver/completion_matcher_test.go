package langserver

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionMatchers(t *testing.T) {
	editRange := lsp.Range{
		Start: lsp.Position{Line: 1, Character: 5},
		End:   lsp.Position{Line: 1, Character: 5},
	}

	cm := NewCompletionMatcher()

	resp := []lsp.CompletionItem{
		{
			Label: "item2",
			Kind:  lsp.CIKFile,
			TextEdit: lsp.TextEdit{
				Range:   editRange,
				NewText: "item2",
			},
		},
	}

	fn := func(r lsp.Range, matched string) ([]lsp.CompletionItem, error) {
		return resp, nil
	}

	err := cm.Register("item", fn)
	require.NoError(t, err)

	list, err := cm.Match(editRange, "local item ")
	require.NoError(t, err)

	assert.Equal(t, resp, list)
}

func TestCompletionMatchers_no_match(t *testing.T) {
	editRange := lsp.Range{
		Start: lsp.Position{Line: 1, Character: 5},
		End:   lsp.Position{Line: 1, Character: 5},
	}

	cm := NewCompletionMatcher()

	resp := []lsp.CompletionItem{
		{
			Label: "item2",
			Kind:  lsp.CIKFile,
			TextEdit: lsp.TextEdit{
				Range:   editRange,
				NewText: "item2",
			},
		},
	}

	fn := func(r lsp.Range, matched string) ([]lsp.CompletionItem, error) {
		return resp, nil
	}

	err := cm.Register("item", fn)
	require.NoError(t, err)

	list, err := cm.Match(editRange, "local foo ")
	require.NoError(t, err)

	expected := []lsp.CompletionItem{}
	assert.Equal(t, expected, list)
}

func TestCompletionMatchers_invalid_term(t *testing.T) {
	cm := NewCompletionMatcher()

	fn := func(r lsp.Range, matched string) ([]lsp.CompletionItem, error) {
		panic("shouldn't be able to get here")
	}

	err := cm.Register("(invalid", fn)
	require.Error(t, err)
}
