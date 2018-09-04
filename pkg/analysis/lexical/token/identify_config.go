package token

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/pkg/errors"
)

// IdentifyConfig is configuration for Identify.
type IdentifyConfig struct {
	path            string
	jsonnetLibPaths []string
	extVar          map[string]string
	extCode         map[string]string
	tlaCode         map[string]string
	tlaVar          map[string]string
}

// NewIdentifyConfig creates an instance of IdentifyConfig.
func NewIdentifyConfig(path string, jPath ...string) (IdentifyConfig, error) {
	ic := IdentifyConfig{
		path:            path,
		jsonnetLibPaths: jPath,
		extCode:         make(map[string]string),
		extVar:          make(map[string]string),
		tlaCode:         make(map[string]string),
		tlaVar:          make(map[string]string),
	}

	dir, file := filepath.Split(ic.path)

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
				return IdentifyConfig{}, err
			}

			// load parameters for current component
			ic.ExtCode("__ksonnet/params", string(data))
		}
	}

	return ic, nil
}

// AddLibPaths adds a path to the Jsonnet lib path.
func (ic *IdentifyConfig) AddLibPaths(path ...string) {
	ic.jsonnetLibPaths = append(ic.jsonnetLibPaths, path...)
}

// ExtCode sets ExtCode.
func (ic *IdentifyConfig) ExtCode(k, v string) {
	ic.extCode[k] = v
}

// ExtVar sets ExtVar.
func (ic *IdentifyConfig) ExtVar(k, v string) {
	ic.extVar[k] = v
}

// TLACode sets TLACode.
func (ic *IdentifyConfig) TLACode(k, v string) {
	ic.tlaCode[k] = v
}

// TLAVar sets TLAVar.
func (ic *IdentifyConfig) TLAVar(k, v string) {
	ic.tlaVar[k] = v
}

// VM create a jsonnet VM using IdentifyConfig.
func (ic *IdentifyConfig) VM() *jsonnet.VM {
	vm := jsonnet.MakeVM()

	importer := &jsonnet.FileImporter{
		JPaths: ic.jsonnetLibPaths,
	}

	vm.Importer(importer)

	for k, v := range ic.extVar {
		vm.ExtVar(k, v)
	}
	for k, v := range ic.extCode {
		vm.ExtCode(k, v)
	}
	for k, v := range ic.tlaCode {
		vm.TLACode(k, v)
	}
	for k, v := range ic.tlaVar {
		vm.TLAVar(k, v)
	}

	return vm
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
