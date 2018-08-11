package lexical

import "github.com/google/go-jsonnet/ast"

type Resolver interface {
	Resolve(ast.Node) error
}
