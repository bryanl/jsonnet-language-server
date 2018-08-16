package locate

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

// Index finds the location of an index.
func Index(idx *ast.Index, l *Locatable, source string) (ast.LocationRange, error) {
	if idx.Id != nil {
		if loc := idx.Loc(); loc != nil {
			id := string(*idx.Id)
			line := loc.Begin.Line
			endCol := loc.End.Column - 1
			beginCol := endCol - len(id) + 1

			r := createRange(loc.FileName, line, beginCol, line, endCol)
			return r, nil
		}

	}

	return ast.LocationRange{}, errors.New("unable to find location for index")
}
