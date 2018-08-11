package jsonneta

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/davecgh/go-spew/spew"
	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

// Position is a position in a document.
type Position struct {
	Line int
	Char int
}

// Describe describes the token at a position.
func Describe(r io.Reader, filename string, p Position, debug bool) (string, error) {
	node, err := snippetToAST(r, filename)
	if err != nil {
		return "", err
	}

	if debug {
		spew.Dump(node)
	}

	defer func() {
		if rec := recover(); rec != nil {
			err = errors.Errorf("recovered from panic: %#v", rec)
		}
	}()

	d := describe(node, p)
	d.pos = p
	d.filename = filename

	return spew.Sdump(d), nil
}

func snippetToAST(r io.Reader, filename string) (ast.Node, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	snippet := string(data)

	return jsonnet.SnippetToAST(filename, snippet)
}

type description struct {
	filename string
	pos      Position
	kind     string
	name     string
}

var blank = &description{
	kind: "noop",
}

func describe(node ast.Node, pos Position) *description {
	if !contains(node, pos) {
		return nil
	}

	switch t := node.(type) {
	default:
		fmt.Printf("unknown node type %T\n", t)
		panic("done")
	case *ast.Local:
		for _, bind := range t.Binds {
			if n := describe(bind.Body, pos); n != nil {
				return n
			}

			if isBefore(bind.Body, pos) {
				return &description{
					kind: "variable",
					name: string(bind.Variable),
				}
			}
		}

		if n := describe(t.Body, pos); n != nil {
			return n
		}

		return blank
	case *ast.Apply:
		if n := describe(t.Target, pos); n != nil {
			return n
		}
		for _, arg := range t.Arguments.Positional {
			if n := describe(arg, pos); n != nil {
				return n
			}
		}
	case *ast.Array:
		for _, element := range t.Elements {
			if n := describe(element, pos); n != nil {
				return n
			}
		}
	case *ast.DesugaredObject:
		for _, field := range t.Fields {
			if n := describe(field.Body, pos); n != nil {
				return n
			}
		}
	case *ast.Index:
		if n := describe(t.Target, pos); n != nil {
			return n
		}
		if n := describe(t.Index, pos); n != nil {
			return n
		}

		return &description{
			kind: "index",
			name: indexValue(t),
		}
	case *ast.Import:
		return &description{
			kind: "import",
			name: t.File.Value,
		}
	case *ast.LiteralString:
		return &description{
			kind: "LiteralString",
			name: t.Value,
		}
	case *ast.Var:
		return &description{
			kind: "Variable",
			name: string(t.Id),
		}
	}

	return blank
}

func indexValue(n *ast.Index) string {
	if n.Index != nil {
		switch t := n.Index.(type) {
		case *ast.LiteralString:
			return t.Value
		}
	}

	if n.Id != nil {
		return string(*n.Id)
	}

	return "unknown"
}

func contains(n ast.Node, pos Position) bool {
	if n.Loc().Begin.Line != n.Loc().End.Line {
		return n.Loc().Begin.Line <= pos.Line &&
			n.Loc().End.Line >= pos.Line
	}

	return n.Loc().Begin.Column <= pos.Char &&
		n.Loc().End.Column >= pos.Char
}

func isBefore(n ast.Node, pos Position) bool {
	if pos.Line == n.Loc().Begin.Line && pos.Char < n.Loc().Begin.Column {
		return true
	} else if pos.Line < n.Loc().Begin.Line {
		return true
	}

	return false
}
