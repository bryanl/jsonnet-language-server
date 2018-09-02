package astext

import "github.com/google/go-jsonnet/ast"

type Partial struct {
	ast.NodeBase
}

type PartialIndex struct {
	ast.NodeBase

	Target ast.Node
}
