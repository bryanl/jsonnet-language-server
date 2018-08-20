package lexical

import (
	"io"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/sirupsen/logrus"
)

type locatableVisitor struct {
	filename string
	visitor  *NodeVisitor
	cache    *LocatableCache
}

func newLocatableVisitor(filename string, r io.Reader, cache *LocatableCache) (*locatableVisitor, error) {
	lv := &locatableVisitor{
		filename: filename,
		cache:    cache,
	}

	v, err := NewNodeVisitor(filename, r, true, PreVisit(lv.previsit))
	if err != nil {
		return nil, err
	}

	lv.visitor = v

	return lv, nil
}

func (lv *locatableVisitor) previsit(token interface{}, parent *locate.Locatable, scope locate.Scope) error {
	r, err := locate.Locate(token, parent, string(lv.visitor.Source))
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

	nl := &locate.Locatable{
		Token:  token,
		Loc:    r,
		Parent: parent,
		Scope:  scope,
	}

	return lv.cache.Store(lv.filename, nl)
}
