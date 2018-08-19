package server

import (
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
)

type complete struct {
	referenceParams lsp.ReferenceParams
	config          *Config
}

func newComplete(rp lsp.ReferenceParams, config *Config) *complete {
	return &complete{
		referenceParams: rp,
		config:          config,
	}
}

func (c *complete) handle() (interface{}, error) {
	uri := c.referenceParams.TextDocument.URI
	text, err := c.config.Text(uri)
	if err != nil {
		return nil, err
	}

	r := strings.NewReader(text)

	path, err := uriToPath(uri)
	if err != nil {
		return nil, err
	}

	loc := posToLoc(c.referenceParams.Position)
	return lexical.CompletionAtLocation(path, r, loc, c.config.JsonnetLibPaths, c.config.NodeCache)
}
