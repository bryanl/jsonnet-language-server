package locate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

const (
	outMostObjectID = "$"
)

// Identifier locates an ast.Identifier.
func Identifier(id ast.Identifier, parentRange ast.LocationRange, source string) (ast.LocationRange, error) {
	if string(id) == outMostObjectID {
		return createRange(parentRange.FileName, 0, 0, 0, 0), nil
	}

	re, err := regexp.Compile(idMatchAssignmentExpr(id))
	if err != nil {
		return ast.LocationRange{}, err
	}

	match := re.FindStringSubmatch(source)
	if len(match) != 3 {
		return ast.LocationRange{}, errors.Errorf("unable to match identifier %q", string(id))
	}

	loc := strings.Index(source, match[0])
	if loc == -1 {
		return ast.LocationRange{}, errors.Errorf("unable to find identifier in source")
	}

	start, err := findLocation(source, loc)
	if err != nil {
		return ast.LocationRange{}, err
	}

	end, err := findLocation(source, loc+len(id)-1)
	if err != nil {
		return ast.LocationRange{}, err
	}

	r := createRange(
		parentRange.FileName,
		start.Line, start.Column,
		end.Line, end.Column,
	)

	return r, nil
}

func idMatchAssignmentExpr(id ast.Identifier) string {
	return fmt.Sprintf(`(?m)(%s)(\(.*?\))?\s*=\s*`, string(id))
}
