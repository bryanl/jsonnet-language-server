package server

import (
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical"
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

	r := strings.NewReader(text)

	path, err := uri.ToPath(uriStr)
	if err != nil {
		return nil, err
	}

	loc := posToLoc(c.referenceParams.Position)
	return lexical.CompletionAtLocation(path, r, loc, c.config)
}
