package locate

import (
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

var (
	// ErrNotLocatable is an error returned when a token has no location.
	ErrNotLocatable = errors.New("not locatable")
)

func Locate(token interface{}, parent *Locatable, source string) (ast.LocationRange, error) {
	var r ast.LocationRange
	var err error

	switch t := token.(type) {
	case *ast.Index:
		r, err = Index(t, parent, source)
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
		return ast.LocationRange{}, errors.Errorf("unable to locate %T", t)
	}

	return r, err
}

type nodeLoc interface {
	Loc() *ast.LocationRange
}
