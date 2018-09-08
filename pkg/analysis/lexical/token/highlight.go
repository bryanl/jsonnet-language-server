package token

import (
	"fmt"

	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
)

// Highlight returns locations to highlight given source and a position.
func Highlight(filepath, source string, pos jpos.Position, nodeCache *NodeCache) ([]jpos.Location, error) {
	node, err := ReadSource(filepath, source, nil)
	if err != nil {
		return nil, err
	}

	sg := scanScope(node, nodeCache)

	found, s, err := sg.at(pos)
	if err != nil {
		return nil, err
	}

	id, path := idNode(found, pos, s)

	return s.refersTo(id, path...), nil
}

func idNode(node ast.Node, pos jpos.Position, s *scope) (ast.Identifier, []string) {
	var id ast.Identifier
	var path []string
	switch found := node.(type) {
	case *ast.DesugaredObject:
		return idNode(s.parent(found), pos, s)
	case *ast.Index:
		v, indexPath := resolveIndex(found)
		id = v.Id
		path = indexPath
	case *ast.Local:
		for _, bind := range found.Binds {
			if pos.IsInJsonnetRange(bind.VarLoc) {
				id = bind.Variable
			}
		}
	case *ast.Var:
		id = found.Id
	default:
		panic(fmt.Sprintf("unable to find nodes of type %T", found))
	}

	return id, path
}
