package lexical

import (
	"io"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/google/go-jsonnet/ast"
	"github.com/sirupsen/logrus"
)

type hoverVisitor struct {
	Visitor *NodeVisitor
	loc     ast.Location

	selectedToken *locate.Locatable
}

func newHoverVisitor(filename string, r io.Reader, loc ast.Location) (*hoverVisitor, error) {
	hv := &hoverVisitor{
		loc: loc,
	}

	logrus.WithFields(logrus.Fields{
		"line":   loc.Line,
		"column": loc.Column,
	}).Info("creating hover visitor")

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

func (hv *hoverVisitor) TokenAtLocation() (*locate.Locatable, error) {
	if err := hv.Visitor.Visit(); err != nil {
		return nil, err
	}

	return hv.selectedToken, nil
}

// previsit figure out bounds for token. If this is not possible, return an error.
// nolint: gocyclo
func (hv *hoverVisitor) previsit(token interface{}, parent *locate.Locatable, env locate.Env) error {
	r, err := locate.Locate(token, parent, string(hv.Visitor.Source))
	if err != nil {
		return err
	}

	if isInvalidRange(r) {
		r = parent.Loc
	}

	name := astext.TokenName(token)
	logrus.Debugf("previsiting %s: %s", name, r.String())

	if r.FileName == "" {
		r.FileName = parent.Loc.FileName
	}

	nl := &locate.Locatable{
		Token:  token,
		Loc:    r,
		Parent: parent,
		Env:    env,
	}

	if hv.selectedToken == nil && inRange(hv.loc, nl.Loc) && nl.Parent != nil {
		logrus.Debugf("setting %T as selected token because there was none (%s)",
			nl.Token, nl.Loc.String())
		hv.selectedToken = nl
	} else if hv.selectedToken != nil && inRange(hv.loc, nl.Loc) && isRangeSmaller(hv.selectedToken.Loc, nl.Loc) {
		logrus.Debugf("setting %T as selected token because its range %s is smaller than %s from %T",
			nl.Token, nl.Loc.String(), hv.selectedToken.Loc.String(), hv.selectedToken.Token)
		hv.selectedToken = nl
	}

	return nil
}

func isInvalidRange(r ast.LocationRange) bool {
	return r.Begin.Line == 0 || r.Begin.Column == 0 &&
		r.End.Line == 0 || r.End.Column == 0
}
