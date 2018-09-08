package token

import (
	"fmt"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
)

type scopeReference struct {
	node ast.Node
	loc  ast.LocationRange
	path []string
}

func (sr *scopeReference) String() string {
	switch n := sr.node.(type) {
	case *ast.Index:
		ls, ok := n.Index.(*ast.LiteralString)
		if !ok {
			panic(fmt.Sprintf("index id type %T", n.Index))
		}
		return fmt.Sprintf("index[%s] - %v", ls.Value, sr.path)
	case *ast.Var:
		return fmt.Sprintf("var[%s] - %v", n.Id, sr.path)
	default:
		panic(fmt.Sprintf("unknown scope reference for type %T", n))
	}
}

type objectLookup struct {
	name string
	path []string
	r    ast.LocationRange
}

type scope struct {
	idMap     map[ast.Identifier]jpos.Location
	declMap   map[ast.Identifier]ast.Node
	refMap    map[ast.Identifier][]scopeReference
	parentMap map[ast.Node]ast.Node
	objectMap map[*ast.DesugaredObject][]objectLookup
	parent    *scope
	nodeCache *NodeCache
}

func newScope2(nodeCache *NodeCache) *scope {
	s := &scope{
		idMap:     make(map[ast.Identifier]jpos.Location),
		declMap:   make(map[ast.Identifier]ast.Node),
		refMap:    make(map[ast.Identifier][]scopeReference),
		objectMap: make(map[*ast.DesugaredObject][]objectLookup),
		parentMap: make(map[ast.Node]ast.Node),
		nodeCache: nodeCache,
	}

	return s
}

func (s *scope) ids() []ast.Identifier {
	var ids []ast.Identifier
	for id := range s.declMap {
		ids = append(ids, id)
	}
	return ids
}

func (s *scope) refersTo(id ast.Identifier, path ...string) []jpos.Location {
	var locations []jpos.Location

	idLoc, ok := s.idMap[id]
	if !ok {
		panic(fmt.Sprintf("couldn't find location of %q", id))
	}

	switch n := s.declMap[id].(type) {
	case *ast.DesugaredObject:
		if len(path) == 0 {
			locations = append(locations, idLoc)
		} else {
			locations = append(locations, s.findObjectPath(n, path)...)
		}

		for _, ref := range s.refMap[id] {
			if slicesEqual(path, ref.path) {
				locations = append(locations, jpos.LocationFromJsonnet(ref.loc))
			}
		}
	default:
		locations = append(locations, idLoc)

		for _, ref := range s.refMap[id] {
			locations = append(locations, jpos.LocationFromJsonnet(ref.loc))
		}
	}

	return locations
}

func (s *scope) findObjectPath(o *ast.DesugaredObject, path []string) []jpos.Location {
	var locations []jpos.Location

	lookups := s.objectMap[o]
	for _, ol := range lookups {
		if slicesEqual(ol.path, path) {
			l := jpos.LocationFromJsonnet(ol.r)
			locations = append(locations, l)
		}
	}

	return locations
}

func (s *scope) declare(id ast.Identifier, loc ast.LocationRange, node ast.Node) {
	s.idMap[id] = jpos.LocationFromJsonnet(loc)
	s.declMap[id] = node
}

func (s *scope) reference(id ast.Identifier, node ast.Node, path ...string) {
	var loc ast.LocationRange
	switch node := node.(type) {
	case *ast.Index:
		ls, ok := node.Index.(*ast.LiteralString)
		if !ok {
			panic(fmt.Sprintf("index id type %T", node.Index))
		}

		loc = *node.Loc()
		loc.Begin.Column = loc.End.Column - len(ls.Value)
	default:
		loc = *node.Loc()
	}

	r := scopeReference{
		node: node,
		path: path,
		loc:  loc,
	}

	if _, ok := s.refMap[id]; !ok {
		s.refMap[id] = make([]scopeReference, 0)
	}

	s.refMap[id] = append(s.refMap[id], r)
}

func (s *scope) indexObject(root, cur *ast.DesugaredObject, name string, path []string) {
	_, ok := s.objectMap[root]
	if !ok {
		s.objectMap[root] = make([]objectLookup, 0)
	}

	ol := objectLookup{
		name: name,
		path: path,
		r:    cur.FieldLocs[name],
	}

	s.objectMap[root] = append(s.objectMap[root], ol)
}

func (s *scope) Clone() *scope {
	clone := &scope{
		idMap:     s.idMap,
		declMap:   make(map[ast.Identifier]ast.Node),
		refMap:    s.refMap,
		objectMap: s.objectMap,
		parentMap: s.parentMap,
		nodeCache: s.nodeCache,
	}

	for k, v := range s.declMap {
		clone.declMap[k] = v
	}

	return clone
}

type scopeGraph struct {
	idScopes   map[ast.Node]*scope
	root       ast.Node
	fieldPath  []string
	rootObject *ast.DesugaredObject
}

func scanScope(node ast.Node, nc *NodeCache) *scopeGraph {
	s := newScope2(nc)

	sg := &scopeGraph{
		idScopes:  make(map[ast.Node]*scope),
		root:      node,
		fieldPath: make([]string, 0),
	}
	sg.visit(nil, node, s)

	return sg
}

func (sg *scopeGraph) at(pos jpos.Position) (*scope, error) {
	n, err := locateNode(sg.root, pos)
	if err != nil {
		return nil, err
	}

	return sg.idScopes[n], nil
}

// nolint: gocyclo
func (sg *scopeGraph) visit(parent, n ast.Node, parentScope *scope) {
	if n == nil {
		return
	}

	currentScope := parentScope
	currentScope.parentMap[n] = parent

	switch n := n.(type) {
	case *ast.Apply:
		sg.visit(n, n.Target, currentScope)
		for _, arg := range n.Arguments.Positional {
			sg.visit(n, arg, currentScope)
		}
		for _, arg := range n.Arguments.Named {
			sg.visit(n, arg.Arg, currentScope)
		}
	case *ast.Array:
		for _, elem := range n.Elements {
			sg.visit(n, elem, currentScope)
		}
	case *ast.Binary:
		sg.visit(n, n.Left, currentScope)
		currentScope.parentMap[n.Right] = n
		sg.visit(n, n.Right, currentScope)
	case *ast.Conditional:
		sg.visit(n, n.Cond, currentScope)
		sg.visit(n, n.BranchTrue, currentScope)
		sg.visit(n, n.BranchFalse, currentScope)
	case *ast.DesugaredObject:
		currentScope = currentScope.Clone()

		inRootObject := false
		ogFieldPath := sg.fieldPath
		if sg.rootObject == nil {
			sg.rootObject = n
			inRootObject = true
		}

		for _, field := range n.Fields {
			name, err := fieldName(field)
			if err != nil {
				continue
			}

			sg.fieldPath = append(sg.fieldPath, name)
			currentScope.indexObject(sg.rootObject, n, name, sg.fieldPath)

			sg.visit(n, field.Name, currentScope)
			sg.visit(n, field.Body, currentScope)
		}

		if inRootObject {
			sg.rootObject = nil
		}
		sg.fieldPath = ogFieldPath
	case *ast.Error:
		currentScope.parentMap[n.Expr] = n
		sg.visit(n, n.Expr, currentScope)
	case *ast.Function:
		currentScope = currentScope.Clone()

		for _, param := range n.Parameters.Required {
			currentScope.declare(param, n.Parameters.RequiredLocs[param], nil)
		}
		for _, param := range n.Parameters.Optional {
			currentScope.declare(param.Name, param.Loc, param.DefaultArg)
		}
		for _, param := range n.Parameters.Optional {
			sg.visit(n, param.DefaultArg, currentScope)
		}
		sg.visit(n, n.Body, currentScope)
	case *ast.Import:
	case *ast.ImportStr:
	case *ast.InSuper:
		sg.visit(n, n.Index, currentScope)
	case *ast.Index:
		v, path := resolveIndex(n)
		currentScope.reference(v.Id, n, path[1:]...)

		sg.visit(n, n.Target, currentScope)
		sg.visit(n, n.Index, currentScope)
	case *ast.LiteralBoolean:
	case *ast.LiteralNull:
	case *ast.LiteralNumber:
	case *ast.LiteralString:
	case *ast.Local:
		currentScope = currentScope.Clone()

		for _, bind := range n.Binds {

			currentScope.declare(bind.Variable, bind.VarLoc, bind.Body)
		}

		for _, bind := range n.Binds {
			sg.visit(n, bind.Body, currentScope)
		}

		sg.visit(n, n.Body, currentScope)
	case *astext.Partial, *astext.PartialIndex:
		// nothing to do
	case *ast.Self:
	case *ast.SuperIndex:
		sg.visit(n, n.Index, currentScope)
	case *ast.Unary:
		sg.visit(n, n.Expr, currentScope)
	case *ast.Var:
		currentScope.reference(n.Id, n)
	default:
		panic(fmt.Sprintf("unexpected node %T", n))
	}

	sg.idScopes[n] = currentScope
}
