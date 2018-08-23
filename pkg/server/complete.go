package server

import (
	"fmt"
	"os"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/langserver"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func matchImport(c *config.Config) langserver.CompletionAction {
	return func(editRange lsp.Range, matched string) ([]lsp.CompletionItem, error) {
		lp := langserver.NewLibPaths(c.JsonnetLibPaths())

		var items []lsp.CompletionItem

		files, err := lp.Files()
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
}

type complete struct {
	referenceParams   lsp.ReferenceParams
	config            *config.Config
	completionMatcher *langserver.CompletionMatcher
}

func defaultMatchers(c *config.Config) map[string]langserver.CompletionAction {
	return map[string]langserver.CompletionAction{
		"import":    matchImport(c),
		"importstr": matchImport(c),
	}
}

func newComplete(rp lsp.ReferenceParams, cfg *config.Config) (*complete, error) {
	c := &complete{
		referenceParams:   rp,
		config:            cfg,
		completionMatcher: langserver.NewCompletionMatcher(),
	}

	for term, fn := range defaultMatchers(cfg) {
		if err := c.completionMatcher.Register(term, fn); err != nil {
			return nil, errors.Wrapf(err, "registering completion match %q", term)
		}
	}

	return c, nil
}

func (c *complete) handle() (interface{}, error) {
	uriStr := c.referenceParams.TextDocument.URI
	text, err := c.config.Text(uriStr)
	if err != nil {
		return nil, errors.Wrap(err, "loading current text")
	}

	spew.Fdump(os.Stderr, "completion reference params",
		c.referenceParams, "===")

	path, err := uri.ToPath(uriStr)
	if err != nil {
		return nil, err
	}

	loc := posToLoc(c.referenceParams.Position)

	list := &lsp.CompletionList{
		Items: []lsp.CompletionItem{},
	}

	pos := c.referenceParams.Position
	editRange := lsp.Range{Start: pos, End: pos}

	logrus.Infof("truncating to %s", loc.String())
	matchText, err := text.Truncate(loc.Line, loc.Column)
	if err != nil {
		return nil, err
	}

	matchItems, err := c.completionMatcher.Match(editRange, matchText)
	if err != nil {
		return nil, err
	}

	if len(matchItems) > 0 {
		return matchItems, nil
	}

	m, err := token.LocationScope(path, text.String(), loc)
	if err != nil {
		logrus.WithError(err).WithField("loc", loc.String()).Debug("load scope")
	} else {
		for _, k := range m.Keys() {
			e, err := m.Get(k)
			if err != nil {
				logrus.WithError(err).Infof("fetching %q from scope", k)
				break
			}

			ci := lsp.CompletionItem{
				Label:         k,
				Kind:          lsp.CIKVariable,
				Detail:        e.Detail,
				Documentation: e.Documentation,
				SortText:      fmt.Sprintf("0_%s", k),
				TextEdit: lsp.TextEdit{
					Range:   lsp.Range{Start: pos, End: pos},
					NewText: k,
				},
			}

			list.Items = append(list.Items, ci)
		}
	}

	for _, k := range m.Keywords() {
		ci := lsp.CompletionItem{
			Label:    k,
			Kind:     lsp.CIKKeyword,
			SortText: fmt.Sprintf("1_%s", k),
			TextEdit: lsp.TextEdit{
				Range:   lsp.Range{Start: pos, End: pos},
				NewText: k,
			},
		}

		list.Items = append(list.Items, ci)
	}

	return list, nil

}
