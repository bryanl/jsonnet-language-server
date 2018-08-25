package server

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/langserver"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_matchHandler_handleImport(t *testing.T) {
	cm := langserver.NewCompletionMatcher()

	td := config.NewTextDocument("file:///file.jsonnet",
		"local foo = {\n    a: \"b\"\n};\n\nlocal y = import ")

	jpm := &fakeJsonnetPathManager{files: []string{"1.jsonnet", "2.libsonnet"}}
	mh := newMatchHandler(jpm, td)
	mh.register(cm)

	pos := position.New(5, 18)
	editRange := position.NewRange(pos, pos)
	got, err := cm.Match(editRange, "local foo = {\n    a: \"b\"\n};\n\nlocal y = import ")
	require.NoError(t, err)

	expected := []lsp.CompletionItem{
		createCompletionItem("1.jsonnet", `"1.jsonnet"`, lsp.CIKFile, editRange, nil),
		createCompletionItem("2.libsonnet", `"2.libsonnet"`, lsp.CIKFile, editRange, nil),
	}
	assert.Equal(t, expected, got)
}

func Test_matchHandler_handleIndex(t *testing.T) {
	cases := []struct {
		name         string
		textDocument config.TextDocument
		at           position.Position
	}{
		{
			name: "handle index",
			textDocument: config.NewTextDocument("file:///file.jsonnet",
				"local o = {\n    a: \"b\"\n};\n\nlocal y = o."),
			at: position.New(5, 13),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cm := langserver.NewCompletionMatcher()

			jpm := &fakeJsonnetPathManager{files: []string{"1.jsonnet", "2.libsonnet"}}
			mh := newMatchHandler(jpm, tc.textDocument)
			mh.register(cm)

			editRange := position.NewRange(tc.at, tc.at)

			// editRange := lsp.Range{Start: lsp.Position{Line: 4, Character: 12}}
			got, err := cm.Match(editRange, "local o = {\n    a: \"b\"\n};\n\nlocal y = o.")
			require.NoError(t, err)

			expected := []lsp.CompletionItem{
				createCompletionItem("a", `a`, lsp.CIKVariable, editRange,
					&token.ScopeEntry{Detail: "o"}),
			}
			assert.Equal(t, expected, got)
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
