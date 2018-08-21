package lexical

import (
	"fmt"
	"io"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/google/go-jsonnet/ast"
	"github.com/sirupsen/logrus"
)

var (
	emptyHover = &lsp.Hover{}
)

func CompletionAtLocation(filename string, r io.Reader, loc ast.Location, cfg *config.Config) (*lsp.CompletionList, error) {
	lc := cfg.LocatableCache()
	l, err := lc.GetAtPosition(filename, loc)
	if err != nil {
		return nil, err
	}

	list := &lsp.CompletionList{
		Items: []lsp.CompletionItem{},
	}

	id := ""
	switch t := l.Token.(type) {
	case *ast.Var:
		id = string(t.Id)
	}

	var ids []string
	for k := range l.Scope {
		ids = append(ids, k)
	}
	logrus.Infof("current scope: %s", strings.Join(ids, ","))

	for k := range l.Scope {
		if strings.HasPrefix(k, id) {
			pos := lsp.Position{
				Line:      loc.Line - 1,
				Character: loc.Column - 1,
			}

			text := strings.TrimPrefix(k, id)

			ci := lsp.CompletionItem{
				Label: k,
				Kind:  lsp.CIKVariable,
				TextEdit: lsp.TextEdit{
					Range:   lsp.Range{Start: pos, End: pos},
					NewText: text,
				},
			}
			list.Items = append(list.Items, ci)
		}
	}

	logrus.WithFields(logrus.Fields{
		"pos":  l.Loc.String(),
		"type": fmt.Sprintf("%T", l.Token),
	}).Infof("found token")

	return list, nil
}

func HoverAtLocation(filename string, r io.Reader, l, c int, cfg *config.Config) (*lsp.Hover, error) {
	loc := ast.Location{
		Line:   l,
		Column: c,
	}

	lc := cfg.LocatableCache()
	locatable, err := lc.GetAtPosition(filename, loc)
	if err != nil {
		return nil, err
	}

	if locatable == nil {
		return emptyHover, nil
	}

	resolved, err := locatable.Resolve(cfg.JsonnetLibPaths(), cfg.NodeCache())
	if err != nil {
		if err == locate.ErrUnresolvable {
			return emptyHover, nil
		}
		return nil, err
	}

	response := &lsp.Hover{
		Contents: []lsp.MarkedString{
			{
				Language: "jsonnet",
				Value:    resolved.Description,
			},
		},
	}

	if hasResolvedLocation(resolved.Location) {
		response.Range = lsp.Range{
			Start: lsp.Position{
				Line:      resolved.Location.Begin.Line - 1,
				Character: resolved.Location.Begin.Column - 1,
			},
			End: lsp.Position{
				Line:      resolved.Location.End.Line - 1,
				Character: resolved.Location.End.Column - 1,
			},
		}
	}

	return response, nil
}

// TODO locatable should own this code
func hasResolvedLocation(r ast.LocationRange) bool {
	locs := []int{r.Begin.Line, r.Begin.Column,
		r.End.Line, r.End.Column}
	for _, l := range locs {
		if l == 0 {
			return false
		}
	}
	return true
}
