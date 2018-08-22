package token

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-jsonnet/ast"
)

func Scope(filename, source string, loc ast.Location) error {
	node, err := Parse(filename, source)
	if err != nil {
		partialNode, isPartial := isPartialNode(err)

		if !isPartial {
			return err
		}

		node = partialNode
	}

	if err = analyze(node); err != nil {
		return err
	}

	spew.Dump(node)

	return nil
}
