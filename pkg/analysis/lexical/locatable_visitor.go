package lexical

import (
	"io"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/google/go-jsonnet/ast"
	"github.com/sirupsen/logrus"
)

type locatableVisitor struct {
	filename   string
	visitor    *NodeVisitor
	locatables []locate.Locatable
}

func newLocatableVisitor(filename string, r io.Reader) (*locatableVisitor, error) {
	lv := &locatableVisitor{
		filename: filename,
	}

	v, err := NewNodeVisitor(filename, r, true, PreVisit(lv.previsit))
	if err != nil {
		return nil, err
	}

	lv.visitor = v

	return lv, nil
}

func (lv *locatableVisitor) Visit() error {
	return lv.visitor.Visit()
}

func (lv *locatableVisitor) Locatables() []locate.Locatable {
	return lv.locatables
}

func (lv *locatableVisitor) previsit(token interface{}, parent *locate.Locatable, scope locate.Scope) error {
	r, err := locate.Locate(token, parent, string(lv.visitor.Source))
	if err != nil {
		return nil
	}

	if isInvalidRange(r) {
		r = parent.Loc
	}

	name := astext.TokenName(token)
	logrus.Debugf("previsiting %s: %s", name, r.String())

	if r.FileName == "" {
		r.FileName = parent.Loc.FileName
	}

	nl := locate.Locatable{
		Token:  token,
		Loc:    r,
		Parent: parent,
		Scope:  scope,
	}

	lv.locatables = append(lv.locatables, nl)
	return nil
}

func isInvalidRange(r ast.LocationRange) bool {
	return r.Begin.Line == 0 || r.Begin.Column == 0 &&
		r.End.Line == 0 || r.End.Column == 0
}
