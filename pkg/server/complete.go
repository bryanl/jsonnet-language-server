package server

import (
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
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
		return nil, err
	}

	path, err := uri.ToPath(uriStr)
	if err != nil {
		return nil, err
	}

	loc := posToLoc(c.referenceParams.Position)

	m, err := token.LocationScope(path, text, loc)
	if err != nil {
		return nil, err
	}

	list := &lsp.CompletionList{
		Items: []lsp.CompletionItem{},
	}

	pos := c.referenceParams.Position

	for _, k := range m.Keys() {
		e, err := m.Get(k)
		if err != nil {
			return nil, err
		}

		ci := lsp.CompletionItem{
			Label:         k,
			Kind:          lsp.CIKVariable,
			Detail:        e.Detail,
			Documentation: e.Documentation,
			TextEdit: lsp.TextEdit{
				Range:   lsp.Range{Start: pos, End: pos},
				NewText: k,
			},
		}

		list.Items = append(list.Items, ci)
	}

	return list, nil

}
