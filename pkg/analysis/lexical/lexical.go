package lexical

import (
	"io"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

// TokenAtLocation returns the token a location in a file.
func TokenAtLocation(filename string, r io.Reader, loc ast.Location) (*Locatable, error) {
	v, err := NewCursorVisitor(filename, r, loc)
	if err != nil {
		return nil, errors.Wrap(err, "create cursor visitor")
	}

	if err = v.Visit(); err != nil {
		return nil, errors.Wrap(err, "visit tokens")
	}

	locatable, err := v.TokenAtPosition()
	if err != nil {
		return nil, errors.Wrap(err, "find token at position")
	}

	return locatable, nil
}
