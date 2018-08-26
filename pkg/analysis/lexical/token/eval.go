package token

import (
	"fmt"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
)

type evaluator struct {
	nodeCache *NodeCache
	until     ast.Node
	scope     *evalScope
	err       error
}

type evalScope struct {
	nodeCache *NodeCache
	store     map[ast.Identifier]ast.Node
}

func newEvalScope(nc *NodeCache) *evalScope {
	return &evalScope{
		store:     make(map[ast.Identifier]ast.Node),
		nodeCache: nc,
	}
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

func (e *evalScope) Clone() *evalScope {
	clone := newEvalScope(e.nodeCache)
	for k, v := range e.store {
		clone.store[k] = v
	}

	return clone
}

// nolint: gocyclo
func (e *evaluator) eval(n ast.Node, parentScope *evalScope) {
	if e.err != nil {
		return
	}

	switch n := n.(type) {
	case *ast.Array:
		for _, elem := range n.Elements {
			e.eval(elem, parentScope)
		}
	case *ast.Apply:
		e.eval(n.Target, parentScope)
	case *ast.Binary:
		e.eval(n.Left, parentScope)
		e.eval(n.Right, parentScope)
	case *ast.Conditional:
		e.eval(n.Cond, parentScope)
		e.eval(n.BranchTrue, parentScope)
		e.eval(n.BranchFalse, parentScope)
	case *ast.DesugaredObject:
		s := parentScope.Clone()
		for _, field := range n.Fields {
			e.eval(field.Name, s)
			e.eval(field.Body, s)
		}
	case *ast.Error:
		e.eval(n.Expr, parentScope)
	case *ast.Function:
		s := parentScope.Clone()

		for _, param := range n.Parameters.Required {
			s.set(param, nil)
		}
		for _, param := range n.Parameters.Optional {
			s.set(param.Name, nil)
		}
		for _, param := range n.Parameters.Optional {
			e.eval(param.DefaultArg, s)
		}
		e.eval(n.Body, s)
	case *ast.Import:
	case *ast.ImportStr:
	case *ast.Index:
		e.eval(n.Target, parentScope)
		e.eval(n.Index, parentScope)
	case *ast.InSuper:
		e.eval(n.Index, parentScope)
	case *ast.LiteralBoolean:
	case *ast.LiteralNull:
	case *ast.LiteralNumber:
	case *ast.LiteralString:
	case *ast.Local:
		s := parentScope.Clone()

		for _, bind := range n.Binds {
			e.err = s.set(bind.Variable, bind.Body)
			e.eval(bind.Body, s)
		}

		e.eval(n.Body, s)
	case *astext.Partial:
		// nothing to do
	case *ast.Self:
	case *ast.SuperIndex:
		e.eval(n.Index, parentScope)
	case *ast.Unary:
		e.eval(n.Expr, parentScope)
	case *ast.Var:
		// nothing to do
	default:
		panic(fmt.Sprintf("unexpected node %T", n))
	}

	if n == e.until {
		e.scope = parentScope
	}
}

func eval(node, until ast.Node, nc *NodeCache) (*evalScope, error) {
	e := evaluator{
		nodeCache: nc,
		until:     until,
		scope:     newEvalScope(nc),
	}

	s := newEvalScope(nc)
	e.eval(node, s)

	if e.err != nil {
		return nil, e.err
	}

	return e.scope, nil
}
