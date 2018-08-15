package locate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	outMostObjectID = "$"
)

// Identifier locates an ast.Identifier.
func Identifier(id ast.Identifier, parent *Locatable, source string) (ast.LocationRange, error) {
	if parent == nil {
		return ast.LocationRange{}, errors.New("parent is nil")
	}

	switch t := parent.Token.(type) {
	case *ast.Index:
		return idInIndex(id, parent, source)
	case ast.LocalBind:
		return idInLocalBind(id, parent.Loc, source)
	case ast.ObjectField:
		return idInObjectField(id, parent, source)
	case ast.ForSpec:
		return idInForSpec(id, parent, source)
	default:
		return ast.LocationRange{}, errors.Errorf("can't locate id in %T", t)
	}
}

func idInForSpec(id ast.Identifier, parent *Locatable, source string) (ast.LocationRange, error) {
	m, err := token.NewMatch(parent.Loc.FileName, source)
	if err != nil {
		return ast.LocationRange{}, err
	}

	logrus.Debugf("looking for `for` at %s", parent.Loc.String())
	pos, err := m.Find(parent.Loc.Begin, token.TokenFor)
	if err != nil {
		return ast.LocationRange{}, err
	}

	t := m.Tokens[pos+1]
	r := createRange(parent.Loc.FileName,
		t.Loc.Begin.Line, t.Loc.Begin.Column,
		t.Loc.End.Line, t.Loc.End.Column)

	return r, nil

}

func idInIndex(id ast.Identifier, parent *Locatable, source string) (ast.LocationRange, error) {
	parentSource, err := extractRange(source, parent.Loc)
	if err != nil {
		return ast.LocationRange{}, err
	}

	tokens, err := Lex("", parentSource)
	if err != nil {
		return ast.LocationRange{}, err
	}

	for i := 1; i < len(tokens)-1; i++ {
		if tokens[i-1].Kind == TokenDot && tokens[i].Data == string(id) {
			r := tokens[i].Loc
			r.Begin.Line += parent.Loc.Begin.Line - 2
			r.Begin.Column += parent.Loc.Begin.Column - 1
			r.End.Line += parent.Loc.Begin.Line - 2
			return r, nil
		}
	}

	return ast.LocationRange{}, errors.New("index not found")
}

func idInObjectField(id ast.Identifier, parent *Locatable, source string) (ast.LocationRange, error) {
	r, err := fieldIDRange(string(id), source)
	if err != nil {
		return ast.LocationRange{}, err
	}

	return r, nil
}

func isZeroRange(r ast.LocationRange) bool {
	return r.Begin.Line == 0 || r.Begin.Column == 0 &&
		r.End.Line == 0 || r.End.Column == 0
}

func idInLocalBind(id ast.Identifier, parentRange ast.LocationRange, source string) (ast.LocationRange, error) {
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
