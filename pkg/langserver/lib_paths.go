package langserver

import (
	"io/ioutil"
	"path/filepath"
	"sort"
)

// LibPaths manage jsonnet lib paths.
type LibPaths struct {
	paths []string
}

// NewLibPaths creates an instance of LibPaths.
func NewLibPaths(paths []string) *LibPaths {
	lp := &LibPaths{
		paths: paths,
	}

	return lp
}

// Files returns files on the jsonnet lib path.
func (lp *LibPaths) Files() ([]string, error) {
	m := make(map[string]bool)

	for _, path := range lp.paths {
		fis, err := ioutil.ReadDir(path)
		if err != nil {
			return nil, err
		}

		for _, fi := range fis {
			if fi.IsDir() {
				continue
			}

			if isJsonnetFile(fi.Name()) {
				m[fi.Name()] = true
			}
		}
	}

	var files []string
	for k := range m {
		files = append(files, k)
	}

	sort.Strings(files)

	return files, nil
}

func isJsonnetFile(name string) bool {
	if ext := filepath.Ext(name); ext == ".jsonnet" || ext == ".libsonnet" {
		return true
	}

	return false
}
