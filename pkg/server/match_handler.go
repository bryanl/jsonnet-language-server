package server

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/langserver"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
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

func (mh *matchHandler) handleImport(editRange position.Range, source, matched string) ([]lsp.CompletionItem, error) {
	logrus.Printf("handling import")
	var items []lsp.CompletionItem

	files, err := mh.jsonnetPathManager.Files()
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		text := fmt.Sprintf(`"%s"`, file)
		ci := createCompletionItem(file, text, lsp.CIKFile, editRange, nil)
		items = append(items, ci)

	}

	return items, nil
}

func (mh *matchHandler) handleIndex(editRange position.Range, source, matched string) ([]lsp.CompletionItem, error) {
	logrus.Printf("handling index")
	loc := editRange.Start

	filename, err := mh.textDocument.Filename()
	if err != nil {
		return nil, err
	}

	var items []lsp.CompletionItem

	scope, err := token.LocationScope(filename, mh.textDocument.String(), loc)
	if err != nil {
		return nil, err
	}

	path, err := resolveIndex(source)
	if err != nil {
		return nil, err
	}

	se, err := scope.GetInPath(path)
	if err != nil {
		return nil, err
	}

	switch n := se.Node.(type) {
	case *ast.DesugaredObject:
		for _, field := range n.Fields {
			name := astext.TokenValue(field.Name)
			ci := createCompletionItem(name, name, lsp.CIKVariable, editRange, se)
			items = append(items, ci)
		}
	}

	return items, nil
}

func createCompletionItem(label, text string, kind int, r position.Range, se *token.ScopeEntry) lsp.CompletionItem {
	var detail, documentation string
	if se != nil {
		detail = se.Detail
		documentation = se.Documentation
	}

	return lsp.CompletionItem{
		Label:         label,
		Kind:          kind,
		Detail:        detail,
		Documentation: documentation,
		TextEdit: lsp.TextEdit{
			Range:   r.ToLSP(),
			NewText: text,
		},
	}
}

var (
	reIndex = regexp.MustCompile(`((\w+\.)*\w+)\.$`)
)

func resolveIndex(source string) ([]string, error) {
	match := reIndex.FindAllString(source, 1)
	if match == nil {
		return nil, errors.Errorf("%q is not part of an index")
	}

	if len(match) != 1 {
		return nil, errors.Errorf("expected only one match when looking for index")
	}

	return removeEmpty(strings.Split(match[0], ".")), nil
}

func removeEmpty(sl []string) []string {
	var out []string
	for _, s := range sl {
		if s != "" {
			out = append(out, s)
		}
	}

	return out
}
