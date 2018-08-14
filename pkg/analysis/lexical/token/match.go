package token

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

// Match matches tokens in a list.
type Match struct {
	Tokens Tokens
}

// NewMatch creates an instance of Match.
func NewMatch(filename, source string) (*Match, error) {
	tokens, err := Lex(filename, source)
	if err != nil {
		return nil, err
	}

	m := &Match{
		Tokens: tokens,
	}

	return m, nil
}

// Bind returns the tokens in a bind.
func (m *Match) Bind(loc ast.Location, name string) (Tokens, error) {
	printTokens(m.Tokens)

	// find a local that starts at `start`
	pos, err := m.Find(loc, TokenLocal)
	if err != nil {
		return nil, err
	}

	begin, end, err := m.bind(pos + 1)
	if err != nil {
		return nil, err
	}

	return m.Tokens[begin : end+1], nil
}

func (m *Match) bind(pos int) (int, int, error) {
	// bind is:
	// 1. id = expr
	// 2. id([params]) = expr

	if m.kind(pos) == TokenIdentifier {
		if m.kind(pos+1) == TokenParenL {
			fmt.Println("bind is a fn")
			return 0, 0, errors.New("bind with fn not implemented")
		} else if m.kind(pos+1) == TokenOperator && m.data(pos+1) == "=" {
			end, err := m.Expr(pos + 2)
			if err != nil {
				return 0, 0, err
			}

			return pos, end, nil
		}
	}

	return 0, 0, errors.New("position is not a bind")
}

// Find finds a token by kind at a position.
func (m *Match) Find(start ast.Location, kind TokenKind) (int, error) {
	for i, t := range m.Tokens {
		if isLocEqual(start, t.Loc.Begin) && kind == t.Kind {
			return i, nil
		}
	}

	return 0, errors.Errorf("not found")
}

// ErrExprNotMatched is an expression is not matched error.
var ErrExprNotMatched = errors.New("expr not matched")

// IsNotMatched returns true if the error is an expression not matched error.
func IsNotMatched(err error) bool {
	return err == ErrExprNotMatched
}

// Expr returns the ending position of an expression started at pos.
func (m *Match) Expr(pos int) (int, error) {
	t := m.Tokens[pos]

	switch t.Kind {
	case TokenNullLit, TokenTrue, TokenFalse, TokenSelf, TokenDollar, TokenStringBlock,
		TokenStringDouble, TokenStringSingle, TokenVerbatimStringDouble,
		TokenVerbatimStringSingle, TokenNumber:
		return pos, nil
	case TokenBracketL:
		if m.kind(pos+1) == TokenBracketR {
			// empty array
			return pos + 1, nil
		}

		cur := pos + 1
		for {
			var err error
			cur, err = m.Expr(cur)
			if err != nil {
				return 0, err
			}

			if m.kind(cur+1) == TokenComma {
				if m.kind(cur+2) == TokenBracketR {
					return cur + 2, nil
				}

				cur = cur + 2
				continue
			} else if m.kind(cur+1) == TokenBracketR {
				return cur + 1, nil
			}

			return 0, errors.New("expected , after expression")
		}
	case TokenIdentifier:
		next := m.Tokens[pos+1]
		if next.Kind == TokenDot {
			end, err := m.Expr(pos + 2)
			if err != nil {
				return 0, err
			}

			return end, nil
		} else if next.Kind == TokenBracketL {
			return m.handleSliceOperator(pos + 1)
		} else if next.Kind == TokenParenL {
			if m.kind(pos+2) == TokenParenR {
				// no args
				return pos + 2, nil
			}
		}

		return pos, nil
	case TokenSuper:
		if t := m.Tokens[pos+1]; t.Kind == TokenDot {
			if m.kind(pos+2) == TokenIdentifier {
				return pos + 2, nil
			}
		} else if m.kind(pos+1) == TokenBracketL {
			fmt.Println("doing the super [] thing")
			end, err := m.Expr(pos + 2)
			if err != nil {
				return 0, err
			}

			if m.kind(end+1) == TokenBracketR {
				return end + 1, nil
			}
		}
	}

	return 0, ErrExprNotMatched
}

// Objlocal returns the ending position of an object local started at pos.
func (m *Match) Objlocal(pos int) (int, error) {
	if m.kind(pos) == TokenLocal {
		_, end, err := m.bind(pos + 1)
		if err != nil {
			return 0, err
		}

		return end, nil
	}
	return 0, errors.New("did not match object local")
}

// Assert returns the ending position of an assert started at pos.
func (m *Match) Assert(pos int) (int, error) {
	if m.kind(pos) == TokenAssert {
		end, err := m.Expr(pos + 1)
		if err != nil {
			return 0, err
		}

		if m.kind(end+1) == TokenOperator && m.data(end+1) == ":" {
			msgEnd, err := m.Expr(end + 1)
			if err != nil {
				return 0, err
			}

			return msgEnd, nil
		}

		return end, nil
	}

	return 0, errors.New("did not match assert")
}

// Fieldname returns the ending position of field name starting at pos.
func (m *Match) Fieldname(pos int) (int, error) {
	if m.kind(pos) == TokenIdentifier {
		return pos, nil
	} else if isString(m.Tokens[pos]) {
		return pos, nil
	} else if m.kind(pos) == TokenBracketL {
		end, err := m.Expr(pos + 1)
		if err != nil {
			return 0, err
		}

		if m.kind(end+1) == TokenBracketR {
			return end + 1, nil
		}
	}

	return 0, errors.New("did not match a field name")
}

// Params returns the ending position of params starting at pos.
func (m *Match) Params(pos int) (int, error) {
	inOptional := false
	for cur := pos; cur < len(m.Tokens)-1; cur++ {
		if m.kind(cur) != TokenIdentifier {
			return 0, errors.Errorf("expected an identifier at %d", cur)
		}

		if m.kind(cur+1) == TokenComma {
			if inOptional {
				return 0, errors.New("required parameter after optional")
			}
			// found required parameter
			cur = cur + 1
		} else if m.kind(cur+1) == TokenOperator && m.data(cur+1) == "=" {
			inOptional = true
			end, err := m.Expr(cur + 2)
			if err != nil {
				return 0, err
			}
			cur = end
			if m.kind(cur+1) == TokenComma {
				cur = cur + 1
			}
		}

		if m.kind(cur+1) == TokenParenR {
			// found end of parameters
			return cur, nil
		}
	}

	return 0, errors.New("did not match parameters")
}

// handleSliceOperator finds the ending position of a slice
// handles the following:
// * [x]
// * [x:x]
// * [x:x:x]
// * [:x]
// * [:x:x]
func (m *Match) handleSliceOperator(pos int) (int, error) {
	handleSliceExtras := func(pos int) (int, error) {
		stopEnd, err := m.Expr(pos)
		if err != nil {
			return 0, err
		}

		if m.kind(stopEnd+1) == TokenBracketR {
			return stopEnd + 1, nil
		} else if isSliceSeperator(m.Tokens[stopEnd+1]) {
			incEnd, err := m.Expr(stopEnd + 2)
			if err != nil {
				return 0, err
			}

			if m.kind(incEnd+1) == TokenBracketR {
				return incEnd + 1, nil
			}
		}

		return 0, errors.New("expected ] after expression")
	}

	if isSliceSeperator(m.Tokens[pos+1]) {
		return handleSliceExtras(pos + 2)
	}
	startEnd, err := m.Expr(pos + 1)
	if err != nil {
		return 0, err
	}

	if m.kind(startEnd+1) == TokenBracketR {
		return startEnd + 1, nil
	} else if isSliceSeperator(m.Tokens[startEnd+1]) {
		return handleSliceExtras(startEnd + 2)
	}

	return 0, errors.New("expected ] after expression")
}

func (m *Match) kind(pos int) TokenKind {
	return m.Tokens[pos].Kind
}

func (m *Match) data(pos int) string {
	return m.Tokens[pos].Data
}

func isLocEqual(l1, l2 ast.Location) bool {
	return l1.Line == l2.Line && l1.Column == l2.Column
}

func isString(t Token) bool {
	switch t.Kind {
	case TokenStringBlock, TokenStringDouble, TokenStringSingle, TokenVerbatimStringDouble,
		TokenVerbatimStringSingle:
		return true
	}

	return false
}

func isSliceSeperator(t Token) bool {
	return t.Kind == TokenOperator && t.Data == ":"
}

func printTokens(tokens Tokens) {
	for i, t := range tokens {
		fmt.Printf("%d %s: %s = %s\n", i, t.Loc.String(), t.Kind.String(), t.Data)
	}
}
