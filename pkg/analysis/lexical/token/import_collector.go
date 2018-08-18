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
	libPath []string
}

// NewImportCollector creates an instance of ImportCollector.
func NewImportCollector(libPath []string) *ImportCollector {
	return &ImportCollector{
		libPath: libPath,
	}
}

// Collect collects imports for a file
func (ic *ImportCollector) Collect(filename string) ([]string, error) {
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

			path, err := ic.importPath(next.Data)
			if err != nil {
				return nil, err
			}

			childPaths, err := ic.Collect(path)
			if err != nil {
				return nil, err
			}

			for _, childPath := range childPaths {
				matches[childPath] = true
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

func (ic *ImportCollector) importPath(filename string) (string, error) {
	for _, dir := range ic.libPath {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		return path, nil
	}

	return "", errors.Errorf("import %q not found in lib path")
}
