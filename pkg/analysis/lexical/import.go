package lexical

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

func ImportSource(paths []string, name string) (ast.Node, error) {
	for _, jPath := range paths {
		sourcePath := filepath.Join(jPath, name)
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			continue
		}

		/* #nosec */
		source, err := ioutil.ReadFile(sourcePath)
		if err != nil {
			return nil, err
		}

		return parse(sourcePath, string(source))
	}

	return nil, errors.Errorf("unable to find import %q", name)
}
