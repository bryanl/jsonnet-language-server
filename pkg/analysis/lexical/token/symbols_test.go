package token

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSymbols(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		expected []Symbol
	}{
		{
			name:   "single",
			source: "local a='a'; a",
			expected: []Symbol{
				{
					name:           "a",
					kind:           lsp.SKString,
					selectionRange: jpos.NewRange(jpos.New(1, 9), jpos.New(1, 12)),
					enclosingRange: jpos.NewRange(jpos.New(1, 9), jpos.New(1, 12)),
				},
			},
		},
		{
			name:   "multiple binds",
			source: "local a=2, b=1; a+b",
			expected: []Symbol{
				{
					name:           "a",
					kind:           lsp.SKNumber,
					selectionRange: jpos.NewRange(jpos.New(1, 9), jpos.New(1, 10)),
					enclosingRange: jpos.NewRange(jpos.New(1, 9), jpos.New(1, 10)),
				},
				{
					name:           "b",
					kind:           lsp.SKNumber,
					selectionRange: jpos.NewRange(jpos.New(1, 14), jpos.New(1, 15)),
					enclosingRange: jpos.NewRange(jpos.New(1, 14), jpos.New(1, 15)),
				},
			},
		},
		{
			name:   "nested",
			source: "local a=1; local b=1; a+1",
			expected: []Symbol{
				{
					name:           "a",
					kind:           lsp.SKNumber,
					selectionRange: jpos.NewRange(jpos.New(1, 9), jpos.New(1, 10)),
					enclosingRange: jpos.NewRange(jpos.New(1, 9), jpos.New(1, 10)),
				},
				{
					name:           "b",
					kind:           lsp.SKNumber,
					selectionRange: jpos.NewRange(jpos.New(1, 20), jpos.New(1, 21)),
					enclosingRange: jpos.NewRange(jpos.New(1, 20), jpos.New(1, 21)),
				},
			},
		},
		{
			name:   "function",
			source: "local id(x) = x; id(1)",
			expected: []Symbol{
				{
					name:           "id",
					kind:           lsp.SKFunction,
					selectionRange: jpos.NewRange(jpos.New(1, 7), jpos.New(1, 16)),
					enclosingRange: jpos.NewRange(jpos.New(1, 7), jpos.New(1, 16)),
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			symbols, err := Symbols(tc.source)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, symbols)
		})
	}
}

func Test_symbolKind(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		expected lsp.SymbolKind
	}{
		{name: "array", source: "[1]", expected: lsp.SKArray},
		{name: "object", source: "{a:1}", expected: lsp.SKObject},
		{name: "function", source: "function() 1", expected: lsp.SKFunction},
		{name: "boolean", source: "true", expected: lsp.SKBoolean},
		{name: "null", source: "null", expected: lsp.SKNull},
		{name: "number", source: "1", expected: lsp.SKNumber},
		{name: "string", source: `"1"`, expected: lsp.SKString},
		{name: "other", source: "1+1", expected: lsp.SKVariable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := ReadSource("source.jsonnet", tc.source, nil)
			require.NoError(t, err)

			got := symbolKind(node)
			assert.Equal(t, tc.expected, got)
		})
	}
}
