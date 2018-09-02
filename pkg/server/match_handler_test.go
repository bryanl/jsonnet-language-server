package server

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/langserver"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_matchHandler_handleImport(t *testing.T) {
	nc := token.NewNodeCache()
	cm := langserver.NewCompletionMatcher()

	source := "local foo = {\n    a: \"b\"\n};\n\nlocal y = import "

	jpm := &fakeJsonnetPathManager{files: []string{"1.jsonnet", "2.libsonnet"}}
	mh := newMatchHandler(jpm, nc)
	mh.register(cm)

	pos := position.New(5, 18)
	got, err := cm.Match(pos, "file.jsonnet", source)
	require.NoError(t, err)

	editRange := position.NewRange(pos, pos)
	expected := []lsp.CompletionItem{
		createCompletionItem("1.jsonnet", `"1.jsonnet"`, lsp.CIKFile, editRange, nil),
		createCompletionItem("2.libsonnet", `"2.libsonnet"`, lsp.CIKFile, editRange, nil),
	}
	assert.Equal(t, expected, got)
}

func Test_matchHandler_handleIndex(t *testing.T) {
	cases := []struct {
		name     string
		text     string
		at       position.Position
		expected func(position.Range) []lsp.CompletionItem
	}{
		{
			name: "handle index",
			text: "local o = {\n    a: \"b\"\n};\n\nlocal y = o.; y",
			at:   position.New(5, 13),
			expected: func(r position.Range) []lsp.CompletionItem {
				return []lsp.CompletionItem{
					createCompletionItem(`"a"`, `"a"`, lsp.CIKVariable, r,
						&token.ScopeEntry{Detail: "(local)"}),
				}
			},
		},
		// {
		// 	name: "nested index",
		// 	text: `local o={data:{a:"a"}};o.data.`,
		// 	at:   position.New(1, 31),
		// 	expected: func(r position.Range) []lsp.CompletionItem {
		// 		return []lsp.CompletionItem{
		// 			createCompletionItem(`"a"`, `"a"`, lsp.CIKVariable, r,
		// 				&token.ScopeEntry{Detail: `(string) "a"`}),
		// 		}
		// 	},
		// },
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nc := token.NewNodeCache()
			cm := langserver.NewCompletionMatcher()

			jpm := &fakeJsonnetPathManager{files: []string{"1.jsonnet", "2.libsonnet"}}
			mh := newMatchHandler(jpm, nc)
			mh.register(cm)

			got, err := cm.Match(tc.at, "file.jsonnet", tc.text)
			require.NoError(t, err)

			editRange := position.NewRange(tc.at, tc.at)
			assert.Equal(t, tc.expected(editRange), got)
		})
	}
}

func OffTest_resolveIndex(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		expected []string
		isErr    bool
	}{
		{
			name:     "incomplete short index",
			in:       "data.",
			expected: []string{"data"},
		},
		{
			name:     "incomplete longer index",
			in:       "o.data.",
			expected: []string{"o", "data"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveIndex(tc.in)
			if tc.isErr {
				require.Error(t, err)
				return
			}

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, got)
			}
		})
	}
}

type fakeJsonnetPathManager struct {
	files    []string
	filesErr error
}

var _ jsonnetPathManager = (*fakeJsonnetPathManager)(nil)

func (jpm *fakeJsonnetPathManager) Files() ([]string, error) {
	return jpm.files, jpm.filesErr
}
