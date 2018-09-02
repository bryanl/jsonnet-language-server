package server

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/pkg/errors"

	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
)

var (
	emptyHover = &lsp.Hover{}
)

type hover struct {
	params lsp.TextDocumentPositionParams
	config *config.Config
	path   string
}

func newHover(params lsp.TextDocumentPositionParams, c *config.Config) (*hover, error) {
	path, err := uri.ToPath(params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	return &hover{
		params: params,
		config: c,
		path:   path,
	}, nil
}

func (h *hover) handle() (interface{}, error) {
	text, err := h.config.Text(h.params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	pos := position.FromLSPPosition(h.params.Position)
	config, err := h.identifyConfig()
	if err != nil {
		return nil, err
	}

	item, err := token.Identify(h.path, text.String(), pos, h.config.NodeCache(), config)
	if err != nil {
		return nil, err
	}

	value := item.String()
	if value == "" {
		return emptyHover, nil
	}

	response := &lsp.Hover{
		Contents: []lsp.MarkedString{
			{
				Language: "jsonnet",
				Value:    value,
			},
		},
	}

	return response, nil
}

func (h *hover) identifyConfig() (token.IdentifyConfig, error) {
	config := token.IdentifyConfig{
		JsonnetLibPaths: h.config.JsonnetLibPaths(),
		ExtCode:         make(map[string]string),
		ExtVar:          make(map[string]string),
		TLACode:         make(map[string]string),
		TLAVar:          make(map[string]string),
	}

	dir, file := filepath.Split(h.path)

	// check to see if this is a ksonnet app
	root, err := findKsonnetRoot(dir)
	if err == nil {
		componentDir := filepath.Join(root, "components")
		if strings.HasPrefix(dir, componentDir) && file != "params.libsonnet" {
			// this file is a component

			// create __ksonnet/params ext code
			paramsFile := filepath.Join(dir, "params.libsonnet")
			data, err := ioutil.ReadFile(paramsFile)
			if err != nil {
				return token.IdentifyConfig{}, err
			}

			config.ExtCode["__ksonnet/params"] = string(data)
		}

		// load parameters for current component
	}

	return config, nil
}

func findKsonnetRoot(cwd string) (string, error) {
	prev := cwd

	for {
		path := filepath.Join(cwd, "app.yaml")
		_, err := os.Stat(path)
		if err == nil {
			return cwd, nil
		}

		if !os.IsNotExist(err) {
			return "", err
		}

		cwd, err = filepath.Abs(filepath.Join(cwd, ".."))
		if err != nil {
			return "", err
		}

		if cwd == prev {
			return "", errors.Errorf("unable to find ksonnet project")
		}

		prev = cwd
	}
}
