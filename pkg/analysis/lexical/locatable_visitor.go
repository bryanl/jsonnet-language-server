package lexical

import (
	"io"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/sirupsen/logrus"
)

type locatableVisitor struct {
	filename   string
	visitor    *NodeVisitor
	locatables []locate.Locatable
}

func newLocatableVisitor(filename string, r io.Reader, diagCh chan<- token.ParseDiagnostic) (*locatableVisitor, error) {
	lv := &locatableVisitor{
		filename: filename,
	}

	v, err := NewNodeVisitor(filename, r, true,
		PreVisit(lv.previsit), parseDiagOpt(diagCh))
	if err != nil {
		return nil, err
	}

	lv.visitor = v

	return lv, nil
}

func parseDiagOpt(diagCh chan<- token.ParseDiagnostic) VisitOpt {
	return func(v *NodeVisitor) {
		v.DiagCh = diagCh
	}
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
