package lexical

import (
	"io"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/google/go-jsonnet/ast"
	"github.com/sirupsen/logrus"
)

type completionVisitor struct {
	Visitor *NodeVisitor
	loc     ast.Location

	selectedToken *locate.Locatable
}

func newCompletionVisitor(filename string, r io.Reader, loc ast.Location) (*completionVisitor, error) {
	cv := &completionVisitor{
		loc: loc,
	}

	logrus.WithFields(logrus.Fields{
		"line":   loc.Line,
		"column": loc.Column,
	}).Info("creating completion visitor")

	v, err := NewNodeVisitor(filename, r, true, PreVisit(cv.previsit))
	if err != nil {
		return nil, err
	}

	cv.Visitor = v

	return cv, nil
}

func (cv *completionVisitor) Visit() error {
	return cv.Visitor.Visit()
}

func (cv *completionVisitor) TokenAtLocation() (*locate.Locatable, error) {
	if err := cv.Visitor.Visit(); err != nil {
		return nil, err
	}

	return cv.selectedToken, nil
}

// previsit figure out bounds for token. If this is not possible, return an error.
// nolint: gocyclo
func (cv *completionVisitor) previsit(token interface{}, parent *locate.Locatable, env locate.Env) error {
	r, err := locate.Locate(token, parent, string(cv.Visitor.Source))
	if err != nil {
		if err == locate.ErrNotLocatable {
			return nil
		}
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

	item := token

	nl := &locate.Locatable{
		Token:  item,
		Loc:    r,
		Parent: parent,
		Env:    env,
	}

	if cv.selectedToken == nil && inRange(cv.loc, nl.Loc) && nl.Parent != nil {
		logrus.Debugf("setting %T as selected token because there was none (%s)",
			nl.Token, nl.Loc.String())
		cv.selectedToken = nl
	} else if cv.selectedToken != nil && inRange(cv.loc, nl.Loc) && isRangeSmaller(cv.selectedToken.Loc, nl.Loc) {
		logrus.Debugf("setting %T as selected token because its range %s is smaller than %s from %T",
			nl.Token, nl.Loc.String(), cv.selectedToken.Loc.String(), cv.selectedToken.Token)
		cv.selectedToken = nl
	}

	return nil
}
