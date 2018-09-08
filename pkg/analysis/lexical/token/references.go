package token

import (
	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

// References finds references to a definition.
func References(path, source string, pos jpos.Position, nodeCache *NodeCache) ([]jpos.Location, error) {
	node, err := ReadSource(path, source, nil)
	if err != nil {
		return nil, err
	}

	found, err := locateNode(node, pos)
	if err != nil {
		return nil, err
	}

	switch n := found.(type) {
	case *ast.DesugaredObject:
		return objectReferences(node, n, pos, nodeCache)
	case *ast.Local:
		return localReferences(node, n, pos, nodeCache)
	default:
		return []jpos.Location{}, nil
	}
}

func localReferences(node ast.Node, local *ast.Local, pos jpos.Position, nodeCache *NodeCache) ([]jpos.Location, error) {
	es, err := eval(node, local, nodeCache)
	if err != nil {
		return nil, err
	}

	for _, bind := range local.Binds {
		if pos.IsInJsonnetRange(bind.VarLoc) {
			var locations []jpos.Location

			localLoc := *local.Loc()
			bodyLoc := *bind.Body.Loc()
			bindRange := jpos.NewRange(
				jpos.FromJsonnetLocation(bind.VarLoc.Begin),
				jpos.FromJsonnetLocation(bodyLoc.End))
			bindLoc := jpos.NewLocation(localLoc.FileName, bindRange)
			locations = append(locations, bindLoc)

			references, ok := es.references[bind.Variable]
			if !ok {
				return nil, errors.Errorf("unable to find references for %q",
					string(bind.Variable))
			}

			for _, r := range references {
				switch r.node.(type) {
				case *ast.Var:
					loc := *r.node.Loc()
					l := jpos.NewLocation(loc.FileName, jpos.FromJsonnetRange(loc))
					locations = append(locations, l)
				}
			}

			return locations, nil
		}
	}

	// nothing matched, but there is no error, so return nil
	return nil, nil
}

func objectReferences(node ast.Node, o *ast.DesugaredObject, pos jpos.Position, nodeCache *NodeCache) ([]jpos.Location, error) {
	var locations []jpos.Location
	es, err := eval(node, o, nodeCache)
	if err != nil {
		return nil, err
	}

	id, n, err := findNodeInScope(es, o)
	if err != nil {
		return nil, err
	}

	parentObject, ok := n.(*ast.DesugaredObject)
	if !ok {
		return nil, nil
	}

	op, err := pathToLocation(parentObject, pos)
	if err != nil {
		return nil, err
	}

	locations = append(locations, jpos.NewLocation(o.Loc().FileName, op.loc))

	// find all references with the identifier and the path
	refLocations, err := locateReferences(id, es, op.path)
	if err != nil {
		return nil, err
	}
	locations = append(locations, refLocations...)

	if op.requiredID != nil {
		paramLocations, err := locateReferences(*op.requiredID, es, nil)
		if err != nil {
			return nil, err
		}

		locations = append(locations, paramLocations...)
	}

	return locations, nil
}

func findNodeInScope(es *evalScope, node ast.Node) (ast.Identifier, ast.Node, error) {
	cur := node

	for _, k := range es.keysAsID() {
		if string(k) == "std" {
			continue
		}
	}

	for {
		id, err := es.scopeID(cur)
		if err == nil {
			return id, cur, nil
		}

		cur, err = es.parent(cur)
		if err != nil {
			return ast.Identifier(""), nil, err
		}
	}
}

func slicesEqual(a, b []string) bool {
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func locateReferences(id ast.Identifier, es *evalScope, path []string) ([]jpos.Location, error) {
	var locations []jpos.Location

	references, ok := es.references[id]
	if !ok {
		return nil, errors.Errorf("unable to find references for %q",
			string(id))
	}

	for _, r := range references {
		if !slicesEqual(path, r.path) {
			continue
		}

		switch r.node.(type) {
		case *ast.Var:
			loc := *r.node.Loc()
			l := jpos.NewLocation(loc.FileName, jpos.FromJsonnetRange(loc))
			locations = append(locations, l)
		case *ast.Index:
			loc := *r.node.Loc()
			l := jpos.NewLocation(loc.FileName, jpos.FromJsonnetRange(loc))
			locations = append(locations, l)
		}
	}

	return locations, nil
}
