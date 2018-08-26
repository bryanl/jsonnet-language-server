package token

import (
	"fmt"
	"log"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/static"
	jlspos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
)

// Identify identifies what is at a position.
func Identify(filename, source string, pos jlspos.Position, nodeCache *NodeCache) (fmt.Stringer, error) {
	node, err := Parse(filename, source, nil)
	if err != nil {
		return nil, err
	}

	if err = DesugarFile(&node); err != nil {
		return nil, err
	}

	err = static.Analyze(node)
	if err != nil {
		return nil, err
	}

	found, err := locateNode(node, pos)
	if err != nil {
		return nil, err
	}

	es, err := eval(node, found, nodeCache)
	if err != nil {
		return nil, err
	}

	scope := newScope(nodeCache)
	scope.addEvalScope(es)

	switch n := found.(type) {
	case *ast.Index:
		_, path := resolveIndex(n)
		se, err := scope.GetInPath(path)
		if err != nil {
			return nil, err
		}

		return astext.NewItem(se.Node), nil
	case *ast.Local:
		return identifyLocal(n, pos, nodeCache)
	case *ast.Var:
		return identifyVar(n, es)
	case nil, *ast.Array, *ast.DesugaredObject, *ast.Import,
		*ast.LiteralBoolean, *ast.LiteralNumber, *ast.LiteralString,
		*astext.Partial:
		return IdentifyNoMatch, nil

	default:
		log.Printf("unable to identify %T", n)
	}

	return IdentifyNoMatch, nil
}

var (
	// IdentifyNoMatch is a no match.
	IdentifyNoMatch = &emptyItem{}
)

type emptyItem struct{}

var _ fmt.Stringer = (*emptyItem)(nil)

func (ei *emptyItem) String() string {
	return ""
}

func identifyVar(v *ast.Var, es *evalScope) (fmt.Stringer, error) {
	x, ok := es.store[v.Id]
	if ok {
		return astext.NewItem(x), nil
	}

	return IdentifyNoMatch, nil
}

func identifyLocal(local *ast.Local, pos jlspos.Position, nodeCache *NodeCache) (fmt.Stringer, error) {
	for _, bind := range local.Binds {
		if pos.IsInJsonnetRange(bind.VarLoc) {
			switch n := bind.Body.(type) {
			case *ast.Import:
				ne, err := nodeCache.Get(n.File.Value)
				if err == nil {
					return astext.NewItem(ne.Node), nil
				}
			default:
				return astext.NewItem(n), nil
			}
		}
	}

	return IdentifyNoMatch, nil
}
