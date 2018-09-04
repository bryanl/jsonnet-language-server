package server

import (
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
)

func textDocumentSignatureHelper(r *request, c *config.Config) (interface{}, error) {
	var params lsp.TextDocumentPositionParams
	if err := r.Decode(&params); err != nil {
		return nil, err
	}

	text, err := c.Text(params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	pos := jpos.FromLSPPosition(params.Position)

	sr, err := token.SignatureHelper(text.String(), pos, c.NodeCache())
	if err != nil {
		return nil, err
	}

	si := lsp.SignatureInformation{
		Label:      sr.Label,
		Parameters: []lsp.ParameterInformation{},
	}

	for _, param := range sr.Parameters {
		si.Parameters = append(si.Parameters, lsp.ParameterInformation{Label: param})
	}

	response := &lsp.SignatureHelp{
		Signatures: []lsp.SignatureInformation{si},
	}

	return response, nil
}
