package token

import (
	"fmt"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	jpos "github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/google/go-jsonnet/ast"
)

// Symbol is a symbol in a jsonnet source.
type Symbol struct {
	name           string
	detail         string
	kind           lsp.SymbolKind
	isDeprecated   bool
	enclosingRange jpos.Range
	selectionRange jpos.Range
	children       []Symbol
}

// Name is the symbol name.
func (s *Symbol) Name() string {
	return s.name
}

// Detail is detail for this symbol, e.g. the singautre of a function.
func (s *Symbol) Detail() string {
	return s.detail
}

// Kind is the kind of this symbol.
func (s *Symbol) Kind() lsp.SymbolKind {
	return s.kind
}

// IsDeprecated indicates if this symbol is deprecated.
func (s *Symbol) IsDeprecated() bool {
	return s.isDeprecated
}

// Range is range enclosing the symbol included leading/trailing
// whitespace and comments.
func (s *Symbol) Range() jpos.Range {
	return s.enclosingRange
}

// SelectionRange is the range enclosing the symbol itself.
func (s *Symbol) SelectionRange() jpos.Range {
	return s.selectionRange
}

type symbolVisitor struct{}

func newSymbolVisitor() *symbolVisitor {
	return &symbolVisitor{}
}

// nolint: gocyclo
func (s *symbolVisitor) visit(n ast.Node) []Symbol {
	var syms []Symbol

	switch n := n.(type) {
	case *ast.Array:
		for _, elem := range n.Elements {
			syms = append(syms, s.visit(elem)...)
		}
	case *ast.Apply:
		syms = append(syms, s.visit(n.Target)...)
	case *ast.Binary:
		syms = append(syms, s.visit(n.Left)...)
		syms = append(syms, s.visit(n.Right)...)
	case *ast.Conditional:
		syms = append(syms, s.visit(n.Cond)...)
		syms = append(syms, s.visit(n.BranchTrue)...)
		syms = append(syms, s.visit(n.BranchFalse)...)
	case *ast.DesugaredObject:
		for _, field := range n.Fields {
			syms = append(syms, s.visit(field.Name)...)
			syms = append(syms, s.visit(field.Body)...)
		}
	case *ast.Error:
		syms = append(syms, s.visit(n.Expr)...)
	case *ast.Function:
		for _, param := range n.Parameters.Optional {
			syms = append(syms, s.visit(param.DefaultArg)...)
		}
		syms = append(syms, s.visit(n.Body)...)
	case *ast.Import:
	case *ast.ImportStr:
	case *ast.Index:
		syms = append(syms, s.visit(n.Target)...)
		syms = append(syms, s.visit(n.Index)...)
	case *ast.InSuper:
		syms = append(syms, s.visit(n.Index)...)
	case *ast.LiteralBoolean:
	case *ast.LiteralNull:
	case *ast.LiteralNumber:
	case *ast.LiteralString:
	case *ast.Local:
		for _, bind := range n.Binds {
			if string(bind.Variable) != "$" {
				// TODO figure out the exact ranges. The AST doesn't provide
				sym := Symbol{
					name:           string(bind.Variable),
					kind:           symbolKind(bind.Body),
					selectionRange: jpos.FromJsonnetRange(*bind.Body.Loc()),
					enclosingRange: jpos.FromJsonnetRange(*bind.Body.Loc()),
				}

				syms = append(syms, sym)
			}

			syms = append(syms, s.visit(bind.Body)...)

		}
		syms = append(syms, s.visit(n.Body)...)
	case *astext.Partial, *astext.PartialIndex:
		// nothing to do
	case *ast.Self:
	case *ast.SuperIndex:
		syms = append(syms, s.visit(n.Index)...)
	case *ast.Unary:
		syms = append(syms, s.visit(n.Expr)...)
	case *ast.Var:
		// nothing to do
	default:
		panic(fmt.Sprintf("unexpected node %T", n))
	}

	return syms
}

// Symbols retrieves symbols from source.
func Symbols(source string) ([]Symbol, error) {
	node, err := ReadSource("symbols.jsonnet", source, nil)
	if err != nil {
		return nil, err
	}

	sv := newSymbolVisitor()
	symbols := sv.visit(node)

	return symbols, nil
}

func symbolKind(node ast.Node) lsp.SymbolKind {
	switch node.(type) {
	case *ast.Array:
		return 18
	case *ast.DesugaredObject:
		return 19
	case *ast.Function:
		return 12
	case *ast.LiteralBoolean:
		return 17
	case *ast.LiteralNull:
		return 21
	case *ast.LiteralNumber:
		return 16
	case *ast.LiteralString:
		return 15
	default:
		return 13
	}
}
