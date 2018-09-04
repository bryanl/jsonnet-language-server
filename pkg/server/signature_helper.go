package server

import (
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/sirupsen/logrus"
)

func textDocumentSignatureHelper(r *request, c *config.Config) (interface{}, error) {
	var posParams lsp.TextDocumentPositionParams
	if err := r.Decode(&posParams); err != nil {
		return nil, err
	}

	text, err := c.Text(posParams.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	r.log().WithFields(logrus.Fields{
		"file": posParams.TextDocument.URI,
		"text": text,
		"pos":  posParams.Position,
	}).Info("input")

	response := &lsp.SignatureHelp{
		Signatures: []lsp.SignatureInformation{
			{
				Label:         "label",
				Documentation: "documentation",
				Paramaters: []lsp.ParameterInformation{
					{
						Label: "x",
					},
					{
						Label: "y",
					},
				},
			},
		},
	}

	return response, nil
}

type signatureInformation struct {
}

type signatureHelper struct {
	config *config.Config
	path   string
	pos    position.Position
	text   string
}

func newSignatureHelper(params lsp.TextDocumentPositionParams, c *config.Config) (*signatureHelper, error) {
	text, err := c.Text(params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	path, err := uri.ToPath(params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	sh := &signatureHelper{
		config: c,
		path:   path,
		pos:    position.FromLSPPosition(params.Position),
		text:   text.String(),
	}

	return sh, nil
}

func (h *signatureHelper) handle() (*signatureInformation, error) {
	ic, err := token.NewIdentifyConfig(h.path, h.config.JsonnetLibPaths()...)
	if err != nil {
		return nil, err
	}

	_, err = token.Identify(h.text, h.pos, h.config.NodeCache(), ic)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
