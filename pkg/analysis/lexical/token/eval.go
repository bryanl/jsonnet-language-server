package token

import (
	"fmt"

	rice "github.com/GeertJohan/go.rice"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

type evaluator struct {
	nodeCache *NodeCache
	until     ast.Node
	scope     *evalScope
	err       error
}

type reference struct {
	node ast.Node
	path []string
}

// evalScope is an evaluation scope.
type evalScope struct {
	nodeCache  *NodeCache
	store      map[ast.Identifier]ast.Node
	references map[ast.Identifier][]reference
	parents    map[ast.Node]ast.Node
}

func newEvalScope(nc *NodeCache) (*evalScope, error) {
	std, err := loadStdlib()
	if err != nil {
		return nil, errors.Wrap(err, "load stdlib")
	}

	return &evalScope{
		store: map[ast.Identifier]ast.Node{
			ast.Identifier("std"): std,
		},
		references: make(map[ast.Identifier][]reference),
		parents:    make(map[ast.Node]ast.Node),
		nodeCache:  nc,
	}, nil
}

func (e *evalScope) keys() []string {
	var sl []string
	for k := range e.store {
		sl = append(sl, string(k))
	}
	return sl
}

func (e *evalScope) keysAsID() []ast.Identifier {
	var ids []ast.Identifier
	for k := range e.store {
		ids = append(ids, k)
	}
	return ids
}

func (e *evalScope) set(id ast.Identifier, node ast.Node) error {
	switch node := node.(type) {
	case *ast.Import:
		ne, err := e.nodeCache.Get(string(node.File.Value))
		if err != nil {
			return err
		}

		e.store[id] = ne.Node
	default:
		e.store[id] = node
	}

	return nil
}

func (e *evalScope) parent(n ast.Node) (ast.Node, error) {
	parent, ok := e.parents[n]
	if !ok {
		return nil, errors.Errorf("unable to find parent for a %T", n)
	}

	return parent, nil
}

func (e *evalScope) scopeID(n ast.Node) (ast.Identifier, error) {
	for k, v := range e.store {
		if v == n {
			return k, nil
		}
	}

	return ast.Identifier(""), errors.New("node is not in scope")
}

func (e *evalScope) refersTo(id ast.Identifier, node ast.Node, path ...string) error {
	_, ok := e.store[id]
	if !ok {
		return errors.Errorf("identifier %q was not in scope", string(id))
	}

	r := reference{
		node: node,
		path: path,
	}

	if _, ok = e.references[id]; !ok {
		e.references[id] = make([]reference, 0)
	}

	e.references[id] = append(e.references[id], r)

	return nil
}

func loadStdlib() (ast.Node, error) {
	box, err := rice.FindBox("ext")
	if err != nil {
		return nil, err
	}

	source, err := box.String("std.jsonnet")
	if err != nil {
		return nil, err
	}

	node, err := Parse("std.jsonnet", source, nil)
	if err != nil {
		return nil, err
	}

	if err = DesugarFile(&node); err != nil {
		return nil, err
	}

	return node, nil
}

func (e *evalScope) Clone() *evalScope {
	clone := &evalScope{
		store:      make(map[ast.Identifier]ast.Node),
		references: e.references,
		parents:    e.parents,
		nodeCache:  e.nodeCache,
	}

	for k, v := range e.store {
		clone.store[k] = v
	}

	return clone
}

// nolint: gocyclo
func (e *evaluator) eval(parent, n ast.Node, parentScope *evalScope) {
	if e.err != nil {
		return
	}

	if n == nil {
		return
	}

	parentScope.parents[n] = parent

	switch n := n.(type) {
	case *ast.Array:
		for _, elem := range n.Elements {
			e.eval(n, elem, parentScope)
		}
	case *ast.Apply:
		e.eval(n, n.Target, parentScope)
	case *ast.Binary:
		e.eval(n, n.Left, parentScope)
		parentScope.parents[n.Right] = n
		e.eval(n, n.Right, parentScope)
	case *ast.Conditional:
		e.eval(n, n.Cond, parentScope)
		e.eval(n, n.BranchTrue, parentScope)
		e.eval(n, n.BranchFalse, parentScope)
	case *ast.DesugaredObject:
		s := parentScope.Clone()
		for _, field := range n.Fields {
			e.eval(n, field.Name, s)
			e.eval(n, field.Body, s)
		}
	case *ast.Object:
		s := parentScope.Clone()
		for _, field := range n.Fields {
			e.eval(n, field.Expr1, s)
			e.eval(n, field.Expr2, s)
			e.eval(n, field.Expr3, s)
		}
	case *ast.Error:
		parentScope.parents[n.Expr] = n
		e.eval(n, n.Expr, parentScope)
	case *ast.Function:
		s := parentScope.Clone()

		for _, param := range n.Parameters.Required {
			if err := s.set(param, nil); err != nil {
				e.err = err
				return
			}
		}
		for _, param := range n.Parameters.Optional {
			if err := s.set(param.Name, nil); err != nil {
				e.err = err
				return
			}
		}
		for _, param := range n.Parameters.Optional {
			e.eval(n, param.DefaultArg, s)
		}
		e.eval(n, n.Body, s)
	case *ast.Import:
	case *ast.ImportStr:
	case *ast.Index:
		v, path := resolveIndex(n)
		if err := parentScope.refersTo(v.Id, n, path[1:]...); err != nil {
			e.err = err
			return
		}

		e.eval(n, n.Target, parentScope)
		e.eval(n, n.Index, parentScope)
	case *ast.InSuper:
		e.eval(n, n.Index, parentScope)
	case *ast.LiteralBoolean:
	case *ast.LiteralNull:
	case *ast.LiteralNumber:
	case *ast.LiteralString:
	case *ast.Local:
		s := parentScope.Clone()

		for _, bind := range n.Binds {
			e.err = s.set(bind.Variable, bind.Body)
			if e.err != nil {
				return
			}
		}

		for _, bind := range n.Binds {
			e.eval(n, bind.Body, s)
		}

		e.eval(n, n.Body, s)
	case *astext.Partial, *astext.PartialIndex:
		// nothing to do
	case *ast.Self:
	case *ast.SuperIndex:
		e.eval(n, n.Index, parentScope)
	case *ast.Unary:
		e.eval(n, n.Expr, parentScope)
	case *ast.Var:
		if err := parentScope.refersTo(n.Id, n); err != nil {
			e.err = err
			return
		}
	default:
		panic(fmt.Sprintf("unexpected node %T", n))
	}

	if n == e.until {
		e.scope = parentScope
	}
}

func eval(node, until ast.Node, nc *NodeCache) (*evalScope, error) {
	es, err := newEvalScope(nc)
	if err != nil {
		return nil, errors.Wrap(err, "create eval scope")
	}

	e := evaluator{
		nodeCache: nc,
		until:     until,
		scope:     es,
	}

	e.eval(nil, node, es)

	if e.err != nil {
		return nil, e.err
	}

	return e.scope, nil
}
