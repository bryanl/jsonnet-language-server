package locate

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

func LocalBind(lb ast.LocalBind, parentRange ast.LocationRange, source string) (ast.LocationRange, error) {
	if fn, ok := lb.Body.(*ast.Function); ok {
		// If the local bind has a function body, the range specificed
		// in the body is the correct size, so no more calcuations are
		// needed.
		return *fn.Loc(), nil
	}

	// Determine where the bind begins. Given the code
	// `local node = "node"`;, the bind portion is `node = "node"`.
	// This means the bind is <identifier> = <value><end>

	expr := fmt.Sprintf(`(?m)%s\s*=\s*`, string(lb.Variable))

	re, err := regexp.Compile(expr)
	if err != nil {
		return ast.LocationRange{}, err
	}

	match := re.FindString(source)
	loc := strings.Index(source, match)
	if loc == -1 {
		return ast.LocationRange{}, errors.Errorf("could not locate local bind")
	}

	start, err := findLocation(source, loc)
	if err != nil {
		return ast.LocationRange{}, err
	}

	r := createRange(
		parentRange.FileName,
		start.Line, start.Column,
		lb.Body.Loc().End.Line,
		lb.Body.Loc().End.Column,
	)

	return r, nil
}

func createRange(name string, r1l, r1c, r2l, r2c int) ast.LocationRange {
	return ast.LocationRange{
		FileName: name,
		Begin:    createLoc(r1l, r1c),
		End:      createLoc(r2l, r2c),
	}
}

func createLoc(line, column int) ast.Location {
	return ast.Location{Line: line, Column: column}
}

// FindLocation finds the location count characters in data.
func findLocation(data string, count int) (ast.Location, error) {
	if count == 0 {
		return ast.Location{}, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Split(bufio.ScanRunes)

	c := 1
	l := 1

	i := 1

	for scanner.Scan() {
		t := scanner.Text()

		if t == "\n" {
			c = 1
			l++
		}

		if i == count {
			break
		}

		i++
		c++
	}

	if err := scanner.Err(); err != nil {
		return ast.Location{}, err
	}

	if count != i {
		return ast.Location{}, errors.Errorf("count didn't match index %d vs %d",
			count, i)
	}

	return ast.Location{
		Line:   l,
		Column: c,
	}, nil
}
