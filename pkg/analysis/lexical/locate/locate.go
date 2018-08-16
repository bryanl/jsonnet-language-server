package locate

import (
	"bufio"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func Locate(token interface{}, parent *Locatable, source string) (ast.LocationRange, error) {
	var r ast.LocationRange
	var err error

	switch t := token.(type) {
	case nodeLoc:
		r = *t.Loc()
	case ast.DesugaredObjectField:
		r, err = DesugaredObjectField(t, parent.Loc, source)
	case ast.ForSpec:
		r, err = ForSpec(t, parent, source)
	case ast.Identifier:
		r, err = Identifier(t, parent, source)
	case *ast.Identifier:
		if t == nil {
			return ast.LocationRange{}, errors.New("identifier was nil")
		}

		r, err = Identifier(*t, parent, source)
	case ast.LocalBind:
		r, err = LocalBind(t, parent.Loc, source)
	case ast.NamedParameter:
		r, err = NamedParameter(t, parent.Loc, source)
	case ast.ObjectField:
		r, err = ObjectField(t, parent, source)
	case astext.RequiredParameter:
		r, err = RequiredParameter(t, parent.Loc, source)
	default:
		logrus.Warnf("previsiting an unlocatable %T with parent %T", t, parent.Token)
		return ast.LocationRange{}, errors.Errorf("unable to locate %T", t)
	}

	return r, err
}

type nodeLoc interface {
	Loc() *ast.LocationRange
}

func findLocation2(source string, pos int) (ast.Location, error) {
	row := 1
	col := 1

	scanner := bufio.NewScanner(strings.NewReader(source))
	scanner.Split(bufio.ScanBytes)

	i := 0
	for scanner.Scan() {
		switch t := scanner.Text(); t {
		case "\n":
			row++
			col = 1
		}

		if pos == i {
			return createLoc(row, col), nil
		}

		i++
	}

	if err := scanner.Err(); err != nil {
		return ast.Location{}, err
	}

	return ast.Location{}, errors.New("position was not in source")
}
