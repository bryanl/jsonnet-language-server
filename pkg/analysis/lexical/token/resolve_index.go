package token

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
)

func resolveIndex(i *ast.Index) []string {
	var cur ast.Node = i
	count := 0
	done := false
	var path []string
	for count < 100 && !done {
		switch c := cur.(type) {
		case *ast.Apply:
			cur = c.Target
		case *ast.Index:
			if c.Index != nil {
				s, ok := c.Index.(*ast.LiteralString)
				if ok {
					path = append([]string{s.Value}, path...)
				}

			}
			cur = c.Target
		case *ast.Self:
			path = append([]string{"self"}, path...)
			done = true
		case *ast.Var:
			path = append([]string{string(c.Id)}, path...)
			done = true
		default:
			panic(fmt.Sprintf("unable to resolve a %T in index", c))
		}

		count++
	}

	return path
}
