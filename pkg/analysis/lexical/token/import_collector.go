package token

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
)

// ImportCollector collects imports from a file and its imports
type ImportCollector struct {
	libPaths []string
}

// NewImportCollector creates an instance of ImportCollector.
func NewImportCollector(libPath []string) *ImportCollector {
	return &ImportCollector{
		libPaths: libPath,
	}
}

// Collect collects imports for a file
func (ic *ImportCollector) Collect(filename string, shallow bool) ([]string, error) {
	source, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// get tokens from filename
	tokens, err := Lex("", string(source))
	if err != nil {
		return nil, err
	}

	matches := make(map[string]bool)

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if t.Kind != TokenImport {
			continue
		}

		if i+1 < len(tokens)-1 {
			next := tokens[i+1]

			matches[next.Data] = true
			i++

			path, err := ImportPath(next.Data, ic.libPaths)
			if err != nil {
				return nil, err
			}

			if !shallow {
				childPaths, err := ic.Collect(path, false)
				if err != nil {
					return nil, err
				}

				for _, childPath := range childPaths {
					matches[childPath] = true
				}
			}
		}
	}

	imports := []string{}

	for k := range matches {
		imports = append(imports, k)
	}

	sort.Strings(imports)

	return imports, nil
}

// ImportPath finds the absolute path to an import.
func ImportPath(filename string, libPaths []string) (string, error) {
	for _, libPath := range libPaths {
		path := filepath.Join(libPath, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		return path, nil
	}

	return "", errors.Errorf("import %q not found in lib path", filename)
}
