package token

import (
	"fmt"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-jsonnet/ast"
)

type locationSet struct {
	store map[jpos.Location]bool
}

func (ls *locationSet) Add(l jpos.Location) {
	if ls.store == nil {
		ls.store = make(map[jpos.Location]bool)
	}

	ls.store[l] = true
}

type identifierSet struct {
	store map[ast.Identifier]bool
}

func (is *identifierSet) add(id ast.Identifier) {
	if is.store == nil {
		is.store = make(map[ast.Identifier]bool)
	}

	is.store[id] = true
}

func (is *identifierSet) contains(id ast.Identifier) bool {
	if is.store == nil {
		return false
	}

	_, ok := is.store[id]
	return ok
}

func (is *identifierSet) String() string {
	var keys []string
	for k := range is.store {
		keys = append(keys, string(k))
	}

	return fmt.Sprintf("[%s]", strings.Join(keys, ","))
}

type scopeReference struct {
	parent ast.Node
	node   ast.Node
	loc    ast.LocationRange
	path   []string
}

func (sr *scopeReference) String() string {
	switch n := sr.node.(type) {
	case *ast.Index:
		ls, ok := n.Index.(*ast.LiteralString)
		if !ok {
			panic(fmt.Sprintf("index id type %T", n.Index))
		}
		return fmt.Sprintf("index[%s] - %v at %s", ls.Value, sr.path, sr.loc.String())
	case *ast.Var:
		return fmt.Sprintf("var[%s] - %v at %s", n.Id, sr.path, sr.loc.String())
	default:
		panic(fmt.Sprintf("unknown scope reference for type %T", n))
	}
}

type objectLookup struct {
	name   string
	path   []string
	r      ast.LocationRange
	object *ast.DesugaredObject
}

type objectKey struct {
	object *ast.DesugaredObject
	field  string
}

type scope struct {
	idMap     map[ast.Identifier]jpos.Location
	declMap   map[ast.Identifier]ast.Node
	refMap    map[ast.Identifier][]scopeReference
	parentMap map[ast.Node]ast.Node
	objectMap map[*ast.DesugaredObject][]objectLookup
	om        *objectMapper
	nodeCache *NodeCache
}

func newScope2(nodeCache *NodeCache) *scope {
	s := &scope{
		idMap:     make(map[ast.Identifier]jpos.Location),
		declMap:   make(map[ast.Identifier]ast.Node),
		refMap:    make(map[ast.Identifier][]scopeReference),
		objectMap: make(map[*ast.DesugaredObject][]objectLookup),
		om:        &objectMapper{},
		parentMap: make(map[ast.Node]ast.Node),
		nodeCache: nodeCache,
	}

	return s
}

func (s *scope) refAt(pos jpos.Position) jpos.Locations {
	var curID *ast.Identifier

	// check if reference is an identifier
	for id, refs := range s.refMap {
		for _, ref := range refs {
			if pos.IsInJsonnetRange(ref.loc) {
				curID = &id
			}
		}
	}

	var locations jpos.Locations
	if curID != nil {
		if _, ok := s.refMap[*curID]; ok {
			for _, ref := range s.refMap[*curID] {
				locations.Add(jpos.LocationFromJsonnet(ref.loc))

				switch n := ref.node.(type) {
				case *ast.Index:
					path := resolveIndex(n)
					fmt.Println("resolving index", path[0])
					if path[0] == "self" {
						if o, ok := ref.parent.(*ast.DesugaredObject); ok {
							l, err := s.om.lookup(o, path[1:])
							if err != nil {
								break
							}
							locations.Add(l)
						}
					}
				default:
					fmt.Printf("how do I add additional locations for %T\n", n)
				}
			}
		}
	}

	return locations
}

func (s *scope) parent(node ast.Node) ast.Node {
	p := s.parentMap[node]
	switch p := p.(type) {
	case *ast.Local:
		if len(p.Binds) == 1 && p.Binds[0].Variable == ast.Identifier("$") {
			return s.parent(p)
		}
	}

	return p
}

func (s *scope) declarations() identifierSet {
	var is identifierSet
	for id := range s.declMap {
		is.add(id)
	}
	return is
}

func (s *scope) identify(node ast.Node) (Identity, error) {
	switch node := node.(type) {
	default:
		panic(fmt.Sprintf("unable to identify %T", node))
	}
}

func (s *scope) refersTo(id ast.Identifier, path ...string) []jpos.Location {
	fmt.Printf("finding what refers to %s at %s\n", id, path)
	var locations []jpos.Location

	idLoc, ok := s.idMap[id]
	if !ok {
		return locations
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

	fmt.Println("findObjectPath path:", path)
	for _, ol := range s.objectMap {
		spew.Dump(ol)
	}

	lookups, ok := s.objectMap[o]
	if !ok {
		return locations
	}
	spew.Dump(s.objectMap, path)
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

func (s *scope) reference(id ast.Identifier, parent, node ast.Node, path ...string) {
	fmt.Printf("creating reference %s%s\n", id, path)
	var loc ast.LocationRange
	switch node := node.(type) {
	case *ast.Index:
		switch in := node.Index.(type) {
		case *ast.LiteralString:
			loc = *node.Loc()
			loc.Begin.Column = loc.End.Column - len(in.Value)
		case *ast.Self:
			loc = *node.Loc()
		case *ast.Var:
			return
		}

	default:
		loc = *node.Loc()
	}

	r := scopeReference{
		parent: parent,
		node:   node,
		path:   path,
		loc:    loc,
	}

	fmt.Printf("created sr for %T path%s at %v\n", node, path, loc.String())

	if _, ok := s.refMap[id]; !ok {
		s.refMap[id] = make([]scopeReference, 0)
	}

	s.refMap[id] = append(s.refMap[id], r)
}

func (s *scope) indexObject(cur *ast.DesugaredObject, name string) {
	if err := s.om.add(cur, name); err != nil {
		panic(fmt.Sprintf("add field: %v", err))
	}
}

func (s *scope) Clone() *scope {
	clone := &scope{
		idMap:     s.idMap,
		declMap:   make(map[ast.Identifier]ast.Node),
		refMap:    s.refMap,
		objectMap: s.objectMap,
		om:        s.om,
		parentMap: s.parentMap,
		nodeCache: s.nodeCache,
	}

	for k, v := range s.declMap {
		clone.declMap[k] = v
	}

	return clone
}

type scopeGraph struct {
	idScopes      map[ast.Node]*scope
	root          ast.Node
	currentObject *ast.DesugaredObject
}

func scanScope(node ast.Node, nc *NodeCache) *scopeGraph {
	s := newScope2(nc)

	sg := &scopeGraph{
		idScopes: make(map[ast.Node]*scope),
		root:     node,
	}
	sg.visit(nil, node, s)

	return sg
}

func (sg *scopeGraph) at(pos jpos.Position) (ast.Node, *scope, error) {
	n, err := locateNode(sg.root, pos)
	if err != nil {
		return nil, nil, err
	}

	return n, sg.idScopes[n], nil
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

		parentObject := sg.currentObject
		sg.currentObject = n

		currentScope.declare(ast.Identifier("self"), *n.Loc(), n)

		for _, field := range n.Fields {
			name, err := fieldName(field)
			if err != nil {
				continue
			}

			currentScope.indexObject(n, name)

			sg.visit(n, field.Name, currentScope)
			sg.visit(n, field.Body, currentScope)
		}

		sg.currentObject = parentObject

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
		path := resolveIndex(n)

		refPath := make([]string, 0)
		if len(path) > 1 {
			refPath = path[1:]
		}

		currentScope.reference(ast.Identifier(path[0]), sg.currentObject, n, refPath...)

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
		currentScope.reference(n.Id, parent, n)
	default:
		panic(fmt.Sprintf("unexpected node %T", n))
	}

	sg.idScopes[n] = currentScope
}
