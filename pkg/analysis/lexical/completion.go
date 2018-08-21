package lexical

import (
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type completion struct {
	locatable *locate.Locatable
}

func newCompletion(l *locate.Locatable) (*completion, error) {
	if l == nil {
		return nil, errors.New("locatable is nil")
	}

	c := &completion{
		locatable: l,
	}

	return c, nil
}

func (c *completion) complete(pos ast.Location) ([]lsp.CompletionItem, error) {
	scope := c.locatable.Scope

	logrus.Infof("current scope: %s", strings.Join(scope.Keys(), ","))

	var items []lsp.CompletionItem
	// for k := range scope.Keys() {

	// }

	return items, nil
}
