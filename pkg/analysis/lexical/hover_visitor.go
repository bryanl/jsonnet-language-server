package lexical

import (
	"io"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type hoverVisitor struct {
	Visitor *NodeVisitor

	selectedToken *Locatable
}

func newHoverVisitor(filename string, r io.Reader, loc ast.Location) (*hoverVisitor, error) {
	hv := &hoverVisitor{}

	v, err := NewNodeVisitor(filename, r, PreVisit(hv.previsit))
	if err != nil {
		return nil, err
	}

	hv.Visitor = v

	return hv, nil
}

func (hv *hoverVisitor) Visit() error {
	return hv.Visitor.Visit()
}

func (hv *hoverVisitor) TokenAtLocation() (*Locatable, error) {
	if err := hv.Visitor.Visit(); err != nil {
		return nil, err
	}

	return hv.selectedToken, nil
}

// previsit figure out bounds for token. If this is not possible, return an error.
// nolint: gocyclo
func (hv *hoverVisitor) previsit(token interface{}, parent *Locatable, env Env) error {
	var r ast.LocationRange
	var err error

	switch t := token.(type) {
	case nodeLoc:
		r = *t.Loc()
	case ast.DesugaredObjectField:
		r, err = locate.DesugaredObjectField(t, parent.Loc, string(hv.Visitor.Source))
	case ast.Identifier:
		r, err = locate.Identifier(t, parent.Loc, string(hv.Visitor.Source))
	case ast.LocalBind:
		r, err = locate.LocalBind(t, parent.Loc, string(hv.Visitor.Source))
	case astext.RequiredParameter:
		r, err = locate.RequiredParameter(t, parent.Loc, string(hv.Visitor.Source))
	default:
		logrus.Printf("previsiting an unlocatable %T with parent %T", t, parent.Token)
		return errors.Errorf("unable locate %T", t)
	}

	if err != nil {
		return err
	}

	if isInvalidRange(r) {
		r = parent.Loc
	}

	logrus.Printf("previsiting %T: %s", token, r.String())

	if r.FileName == "" {
		r.FileName = parent.Loc.FileName
	}

	nl := &Locatable{
		Token:  token,
		Loc:    r,
		Parent: parent,
	}

	if hv.selectedToken == nil {
		logrus.Printf("setting %T as selected token because there was none",
			nl.Token)
		hv.selectedToken = nl
	} else if isRangeSmaller(hv.selectedToken.Loc, nl.Loc) {
		logrus.Printf("setting %T as selected token because its range %s is smaller than %s from %T",
			nl.Token, nl.Loc.String(), hv.selectedToken.Loc.String(), hv.selectedToken.Token)
		hv.selectedToken = nl
	}

	return nil
}

type nodeLoc interface {
	Loc() *ast.LocationRange
}

func isInvalidRange(r ast.LocationRange) bool {
	return r.Begin.Line == 0 || r.Begin.Column == 0 &&
		r.End.Line == 0 || r.End.Column == 0
}
