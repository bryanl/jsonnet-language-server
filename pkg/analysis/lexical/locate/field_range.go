package locate

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

func fieldRange(fieldName, source string) (ast.LocationRange, error) {
	tokens, err := Lex("", source)
	if err != nil {
		return ast.LocationRange{}, err
	}

	if len(tokens) < 2 || (tokens[0].Kind != TokenBraceL && tokens[len(tokens)-1].Kind != TokenBraceR) {
		return ast.LocationRange{}, errors.New("source is not an object")
	}

	found := false
	depth := 0

	var r ast.LocationRange

	for i := 1; i < len(tokens)-2; i++ {
		token := tokens[i]

		switch token.Kind {
		case TokenIdentifier:
			if depth == 0 && !found {
				if tokens[i].Data == fieldName {
					if tf, x := scanValue(tokens, i); tf {
						i += x
						found = true
						r.Begin = token.Loc.Begin
						continue
					} else if tf, x := scanFn(tokens, i); tf {
						i += x
						found = true
						r.Begin = token.Loc.Begin
						continue
					}
					found = false
				}
			}
		case TokenBraceL, TokenBracketL:
			depth++
		case TokenBraceR, TokenBracketR:
			depth--
		}

		if depth == 0 && found {
			r.End = token.Loc.End
			return r, nil
		}
	}

	spew.Dump(source)

	return ast.LocationRange{}, errors.New("object is not complete 1")
}

func fieldIDRange(fieldName, source string) (ast.LocationRange, error) {
	tokens, err := Lex("", source)
	if err != nil {
		return ast.LocationRange{}, err
	}

	found := false
	depth := 0

	for i := 0; i < len(tokens)-1; i++ {
		token := tokens[i]

		switch token.Kind {
		case TokenIdentifier:
			if depth == 0 && !found {
				if tokens[i].Data == fieldName {
					if tf, _ := scanValue(tokens, i); tf {
						return token.Loc, nil
					} else if tf, x := scanFn(tokens, i); tf {
						loc := token.Loc
						loc.End.Column += x
						return loc, nil
					}
				}
			}
		}
	}

	return ast.LocationRange{}, errors.Errorf("object is not complete")
}

func scanValue(tokens []Token, pos int) (bool, int) {
	if tokens[pos+1].Kind == TokenOperator {
		return true, 1
	}

	return false, 0
}

func scanFn(tokens []Token, pos int) (bool, int) {
	isFn := tokens[pos+1].Kind == TokenParenL

	if isFn {
		depth := 0

		for i := pos + 1; i < len(tokens); i++ {
			switch tokens[i].Kind {
			case TokenParenL:
				depth++
			case TokenParenR:
				depth--
			default:
				if depth == 0 {
					return true, i - pos
				}
			}
		}
	}

	return false, 0
}
