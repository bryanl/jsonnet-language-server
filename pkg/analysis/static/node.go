package static

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

// AtPosition returns the node at a postion or an error.
func AtPosition(rootNode ast.Node, loc ast.Location) (ast.Node, error) {
	if isAtEnd(rootNode, loc) {
		// loc = rootNode.Loc().End
	}

	// TODO finish me
	return nil, errors.Errorf("not implemented")
}

func isAtEnd(node ast.Node, loc ast.Location) bool {
	endLoc := node.Loc().End
	return endLoc.Line < loc.Line || (endLoc.Line == loc.Line && endLoc.Column < loc.Column)
}
