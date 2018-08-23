package server

import (
	"fmt"
	"os"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type complete struct {
	referenceParams lsp.ReferenceParams
	config          *config.Config
}

func newComplete(rp lsp.ReferenceParams, config *config.Config) *complete {
	return &complete{
		referenceParams: rp,
		config:          config,
	}
}

func (c *complete) handle() (interface{}, error) {
	uriStr := c.referenceParams.TextDocument.URI
	text, err := c.config.Text(uriStr)
	if err != nil {
		return nil, errors.Wrap(err, "loading current text")
	}

	spew.Fdump(os.Stderr, c.referenceParams)

	path, err := uri.ToPath(uriStr)
	if err != nil {
		return nil, err
	}

	loc := posToLoc(c.referenceParams.Position)

	list := &lsp.CompletionList{
		Items: []lsp.CompletionItem{},
	}

	pos := c.referenceParams.Position

	m, err := token.LocationScope(path, text, loc)
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
