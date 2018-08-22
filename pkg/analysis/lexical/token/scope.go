package token

import (
	"github.com/google/go-jsonnet/ast"
)

// Scope finds the free variables for a location.
func Scope(filename, source string, loc ast.Location) (ast.Identifiers, error) {
	node, err := Parse(filename, source)
	if err != nil {
		partialNode, isPartial := isPartialNode(err)

		if !isPartial {
			return nil, err
		}

		node = partialNode
	}

	if err = analyze(node); err != nil {
		return nil, err
	}

	found, err := locate(node, loc)
	if err != nil {
		return nil, err
	}

	return found.FreeVariables(), nil
}
