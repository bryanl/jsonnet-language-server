package token

import (
	"fmt"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
)

type locator struct {
	loc           jpos.Position
	err           error
	enclosingNode ast.Node
}

func (l *locator) visitNext(a ast.Node) {
	if l.err != nil {
		return
	}

	switch a.(type) {
	case *astext.Partial:
		l.enclosingNode = a
		return
	}

	if l.loc.IsInJsonnetRange(*a.Loc()) {
		if l.enclosingNode == nil {
			l.enclosingNode = a
		} else if isRangeSmaller(*l.enclosingNode.Loc(), *a.Loc()) {
			l.enclosingNode = a
		}
	}

	l.err = l.analyzeVisit(a)
}

// nolint: gocyclo
func (l *locator) analyzeVisit(a ast.Node) error {
	switch a := a.(type) {
	case *ast.Apply:
		l.visitNext(a.Target)
		for _, arg := range a.Arguments.Positional {
			l.visitNext(arg)
		}
		for _, arg := range a.Arguments.Named {
			l.visitNext(arg.Arg)
		}
	case *ast.Array:
		for _, elem := range a.Elements {
			l.visitNext(elem)
		}
	case *ast.Binary:
		l.visitNext(a.Left)
		l.visitNext(a.Right)
	case *ast.Conditional:
		l.visitNext(a.Cond)
		l.visitNext(a.BranchTrue)
		l.visitNext(a.BranchFalse)
	case *ast.Error:
		l.visitNext(a.Expr)
	case *ast.Function:
		for _, param := range a.Parameters.Optional {
			l.visitNext(param.DefaultArg)
		}
		l.visitNext(a.Body)
	case *ast.Import:
		//nothing to do here
	case *ast.ImportStr:
		//nothing to do here
	case *ast.InSuper:
		l.visitNext(a.Index)
	case *ast.SuperIndex:
		l.visitNext(a.Index)
	case *ast.Index:
		l.visitNext(a.Target)
		if a.Index != nil {
			l.visitNext(a.Index)
		}
	case *ast.Local:
		for _, bind := range a.Binds {
			l.visitNext(bind.Body)
		}
		l.visitNext(a.Body)
	case *ast.LiteralBoolean:
		//nothing to do here
	case *ast.LiteralNull:
		//nothing to do here
	case *ast.LiteralNumber:
		//nothing to do here
	case *ast.LiteralString:
		//nothing to do here
	case *ast.DesugaredObject:
		for _, field := range a.Fields {
			l.visitNext(field.Name)
			l.visitNext(field.Body)
		}
		for _, assert := range a.Asserts {
			l.visitNext(assert)
		}
	case *ast.Object:
		for _, field := range a.Fields {
			if field.Kind == ast.ObjectFieldExpr ||
				field.Kind == ast.ObjectFieldStr {
				l.visitNext(field.Expr1)
			}

			if field.Expr2 != nil {
				l.visitNext(field.Expr2)
			}

			if field.Expr3 != nil {
				l.visitNext(field.Expr3)
			}
		}
	case *ast.Self:
		//nothing to do here
	case *ast.Unary:
		l.visitNext(a.Expr)
	case *ast.Var:
		//nothing to do here
	case *astext.Partial:
		//nothing to do here
	case *astext.PartialIndex:
		l.visitNext(a.Target)
	case nil:
	default:
		panic(fmt.Sprintf("Unexpected node %#v", a))
	}

	return l.err
}

func locateNode(node ast.Node, pos jpos.Position) (ast.Node, error) {
	if node == nil {
		return &astext.Partial{}, nil
	}

	l := &locator{
		loc:           pos,
		enclosingNode: node,
	}
	if err := l.analyzeVisit(node); err != nil {
		return nil, err
	}
	return l.enclosingNode, nil
}

func isRangeSmaller(r1, r2 ast.LocationRange) bool {
	return beforeRangeOrEqual(r1.Begin, r2) &&
		afterRangeOrEqual(r1.End, r2)
}

func beforeRangeOrEqual(l ast.Location, r ast.LocationRange) bool {
	begin := r.Begin
	if l.Line < begin.Line {
		return true
	} else if l.Line == begin.Line && l.Column <= begin.Column {
		return true
	}

	return false
}

func afterRangeOrEqual(l ast.Location, lr ast.LocationRange) bool {
	end := lr.End
	if l.Line > end.Line {
		return true
	} else if l.Line == end.Line && l.Column >= end.Column {
		return true
	}

	return false
}
