package server

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/langserver"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
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

	editRange := lsp.Range{}
	got, err := cm.Match(editRange, "local foo = {\n    a: \"b\"\n};\n\nlocal y = import ")
	require.NoError(t, err)

	expected := []lsp.CompletionItem{
		{
			Label:    "1.jsonnet",
			Kind:     lsp.CIKFile,
			TextEdit: lsp.TextEdit{NewText: `"1.jsonnet"`, Range: editRange},
		},
		{
			Label:    "2.libsonnet",
			Kind:     lsp.CIKFile,
			TextEdit: lsp.TextEdit{NewText: `"2.libsonnet"`, Range: editRange},
		},
	}
	assert.Equal(t, expected, got)
}

func Test_matchHandler_handleIndex(t *testing.T) {
	cm := langserver.NewCompletionMatcher()

	td := config.NewTextDocument("file:///file.jsonnet",
		"local o = {\n    a: \"b\"\n};\n\nlocal y = o.")

	jpm := &fakeJsonnetPathManager{files: []string{"1.jsonnet", "2.libsonnet"}}
	mh := newMatchHandler(jpm, td)
	mh.register(cm)

	editRange := lsp.Range{Start: lsp.Position{Line: 4, Character: 12}}
	got, err := cm.Match(editRange, "local o = {\n    a: \"b\"\n};\n\nlocal y = o.")
	require.NoError(t, err)

	expected := []lsp.CompletionItem{
		{
			Label:    "a",
			Detail:   "o",
			Kind:     int(lsp.CIKVariable),
			TextEdit: lsp.TextEdit{NewText: `a`, Range: editRange},
		},
	}
	assert.Equal(t, expected, got)
}

type fakeJsonnetPathManager struct {
	files    []string
	filesErr error
}

var _ jsonnetPathManager = (*fakeJsonnetPathManager)(nil)

func (jpm *fakeJsonnetPathManager) Files() ([]string, error) {
	return jpm.files, jpm.filesErr
}
