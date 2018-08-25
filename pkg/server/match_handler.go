package server

import (
	"fmt"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/langserver"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type jsonnetPathManager interface {
	Files() ([]string, error)
}

type defaultJsonnetPathManager struct {
	config *config.Config
}

func newJsonnetPathManager(c *config.Config) *defaultJsonnetPathManager {
	return &defaultJsonnetPathManager{
		config: c,
	}
}

func (jpm *defaultJsonnetPathManager) Files() ([]string, error) {
	lp := langserver.NewLibPaths(jpm.config.JsonnetLibPaths())
	return lp.Files()
}

type matchHandler struct {
	jsonnetPathManager jsonnetPathManager
	textDocument       config.TextDocument
}

func newMatchHandler(jpm jsonnetPathManager, td config.TextDocument) *matchHandler {
	mh := &matchHandler{
		jsonnetPathManager: jpm,
		textDocument:       td,
	}

	return mh
}

func (mh *matchHandler) register(cm *langserver.CompletionMatcher) error {
	m := map[string]langserver.CompletionAction{
		`import\s`:    mh.handleImport,
		`importstr\s`: mh.handleImport,
		`\w+\.`:       mh.handleIndex,
	}

	for term, fn := range m {
		if err := cm.Register(term, fn); err != nil {
			return errors.Wrapf(err, "registering completion matcher %q", term)
		}
	}

	return nil
}

func (mh *matchHandler) handleImport(editRange lsp.Range, matched string) ([]lsp.CompletionItem, error) {
	var items []lsp.CompletionItem

	files, err := mh.jsonnetPathManager.Files()
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		ci := lsp.CompletionItem{
			Label: file,
			Kind:  lsp.CIKFile,
			TextEdit: lsp.TextEdit{
				Range:   editRange,
				NewText: fmt.Sprintf(`"%s"`, file),
			},
		}

		items = append(items, ci)

	}

	return items, nil
}

func (mh *matchHandler) handleIndex(editRange lsp.Range, matched string) ([]lsp.CompletionItem, error) {
	logrus.Printf("handling index")
	loc := posToLoc(editRange.Start)

	filename, err := mh.textDocument.Filename()
	if err != nil {
		return nil, err
	}

	var items []lsp.CompletionItem

	scope, err := token.LocationScope(filename, mh.textDocument.String(), loc)
	if err != nil {
		return nil, err
	}

	varName := strings.TrimSuffix(matched, ".")
	se, err := scope.Get(varName)
	if err != nil {
		return nil, err
	}

	switch n := se.Node.(type) {
	case *ast.DesugaredObject:
		for _, field := range n.Fields {
			name := astext.TokenValue(field.Name)

			ci := lsp.CompletionItem{
				Label:         name,
				Kind:          lsp.CIKVariable,
				Detail:        se.Detail,
				Documentation: se.Documentation,
				TextEdit: lsp.TextEdit{
					Range:   editRange,
					NewText: name,
				},
			}

			items = append(items, ci)
		}
	}

	return items, nil
}
