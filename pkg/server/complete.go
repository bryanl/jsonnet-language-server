package server

import (
	"fmt"
	"os"

	"github.com/bryanl/jsonnet-language-server/pkg/util/position"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/langserver"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type complete struct {
	referenceParams   lsp.ReferenceParams
	config            *config.Config
	completionMatcher *langserver.CompletionMatcher
}

func newComplete(rp lsp.ReferenceParams, cfg *config.Config) (*complete, error) {
	c := &complete{
		referenceParams:   rp,
		config:            cfg,
		completionMatcher: langserver.NewCompletionMatcher(),
	}

	td, err := cfg.Text(rp.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	jpm := newJsonnetPathManager(cfg)
	mh := newMatchHandler(jpm, *td)
	if err := mh.register(c.completionMatcher); err != nil {
		return nil, err
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

	list := &lsp.CompletionList{
		Items: []lsp.CompletionItem{},
	}

	pos := position.FromLSPPosition(c.referenceParams.Position)
	editRange := position.NewRange(pos, pos)

	logrus.Infof("truncating to %s", pos.String())
	matchText, err := text.Truncate(pos)
	if err != nil {
		return nil, err
	}
	logrus.Info(matchText)

	matchItems, err := c.completionMatcher.Match(editRange, matchText)
	if err != nil {
		return nil, err
	}

	if len(matchItems) > 0 {
		return matchItems, nil
	}

	m, err := token.LocationScope(path, text.String(), pos)
	if err != nil {
		logrus.WithError(err).WithField("loc", pos.String()).Debug("load scope")
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
					Range:   editRange.ToLSP(),
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
				Range:   editRange.ToLSP(),
				NewText: k,
			},
		}

		list.Items = append(list.Items, ci)
	}

	return list, nil

}
