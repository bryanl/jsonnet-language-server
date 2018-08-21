package token

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

// Match matches tokens in a list.
type Match struct {
	Tokens []Token
	pos    int
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

// Pos returns the current postion.
func (m *Match) Pos() int {
	return m.pos
}

// Bind returns the tokens in a bind.
func (m *Match) Bind(loc ast.Location, name string) (Tokens, error) {
	// find a local that starts at `start`
	pos, err := m.Find(loc, TokenLocal)
	if err != nil {
		return nil, err
	}
	m.pos = pos + 1

	begin, end, err := m.bind()
	if err != nil {
		return nil, err
	}

	return m.Tokens[begin : end+1], nil
}

func (m *Match) FindObjectField(loc ast.Location, name string) (Tokens, error) {
	objectStartPos, err := m.Find(loc, TokenBraceL)
	if err != nil {
		return nil, err
	}

	for i := objectStartPos + 1; i < m.len(); i++ {
		if m.kind(i) == TokenLocal {
			end, err := m.Objlocal(i)
			if err != nil {
				return nil, err
			}

			if m.kind(i+1) == TokenIdentifier &&
				name == m.data(i+1) {
				return m.Tokens[i : end+1], nil
			}

			if m.hasTrailingComma(end) {
				end++
			}

			i = end

		} else if m.kind(i) == TokenAssert {
			end, err := m.Assert(i)
			if err != nil {
				return nil, err
			}

			if m.hasTrailingComma(end) {
				end++
			}

			i = end
		} else {
			m.pos = i
			found, err := m.findFieldName()
			if err != nil {
				return nil, err
			}

			fieldEndPos, err := m.Field(i)
			if err != nil {
				return nil, err
			}

			if name == found {
				return m.Tokens[i : fieldEndPos+1], nil
			}

			i = fieldEndPos

			if m.hasTrailingComma(i) {
				i++
			}

			if m.kind(i+1) == TokenBraceR {
				return nil, errors.Errorf("was not able to find field %q in object", name)
			}
		}
	}

	return nil, errors.New("object field not found")
}

func (m *Match) findFieldName() (string, error) {
	if m.kind(m.pos) == TokenIdentifier {
		return m.data(m.pos), nil
	} else if m.isString(m.pos) {
		return m.data(m.pos), nil
	} else if m.kind(m.pos) == TokenBracketL {
		return fmt.Sprintf("[%s]", m.data(m.pos+1)), nil
	} else if m.kind(m.pos) == TokenLocal {
		return TokenLocal.String(), nil
	}

	return "", errors.Errorf("invalid field name at %s; token = %s",
		m.Tokens[m.pos].Loc.String(), m.kind(m.pos))
}

func (m *Match) bind() (int, int, error) {
	// bind is:
	// 1. id = expr
	// 2. id([params]) = expr

	if m.kind(m.pos) == TokenIdentifier {
		if m.kind(m.pos+1) == TokenParenL {
			pos, err := m.Params(m.pos + 2)
			if err != nil {
				return 0, 0, err
			}

			if m.kind(pos+1) != TokenParenR {
				return 0, 0, errors.New("a ')' was expected")
			}
			return m.pos, pos, nil
		} else if m.isOperator(m.pos+1, "=") {
			end, err := m.Expr(m.pos + 2)
			if err != nil {
				return 0, 0, err
			}

			return m.pos, end, nil
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

	return 0, errors.Errorf("token %q at %s was not found",
		kind.String(), start.String())
}

// FindBeforeLocation finds the token before a location.
func (m *Match) FindBeforeLocation(loc ast.Location) (int, error) {
	for i, t := range m.Tokens {
		if loc.Line == t.Loc.Begin.Line {
			if t.Loc.End.Column == loc.Column {
				return i, nil
			}
		}
	}

	return 0, errors.Errorf("unable to find token at %s", loc.String())
}

func (m *Match) FindFirst(start ast.Location, kind TokenKind) (int, error) {
	for i, t := range m.Tokens {
		if start.Line > t.Loc.Begin.Line {
			continue
		} else if start.Line == t.Loc.End.Line &&
			start.Column > t.Loc.End.Column {
			continue
		}

		if t.Kind == kind {
			return i, nil
		}
	}

	return 0, errors.Errorf("token %q was not found", kind.String())
}

// ErrExprNotMatched is an expression is not matched error.
var ErrExprNotMatched = errors.New("expr not matched")

// IsNotMatched returns true if the error is an expression not matched error.
func IsNotMatched(err error) bool {
	return err == ErrExprNotMatched
}

// Expr returns the ending position of an expression started at pos.
func (m *Match) Expr(pos int) (int, error) {
	end, err := m.expr(pos)
	if err != nil {
		return 0, err
	}

	if m.kind(end+1) == TokenParenL {
		end, err = m.Params(end + 2)
		if err != nil {
			return 0, err
		}

		if m.kind(end+1) != TokenParenR {
			return 0, errors.New("expeding ')'")
		}

		end = end + 1
	} else if m.kind(end+1) == TokenIn && m.kind(end+2) == TokenSuper {
		end = end + 2
	} else if m.kind(end+1) == TokenOperator && isBinaryOp(m.data(end+1)) {
		end, err = m.Expr(end + 2)
		if err != nil {
			return 0, err
		}
	}

	return end, nil
}

// nolint: gocyclo
func (m *Match) expr(pos int) (int, error) {
	if pos > len(m.Tokens)-1 {
		return 0, errors.New("position overflows tokens")
	}

	if m.isKind(pos, []TokenKind{TokenComma}) {
		pos++
	}

	t := m.Tokens[pos]

	switch t.Kind {
	case TokenNullLit, TokenTrue, TokenFalse, TokenSelf, TokenDollar, TokenStringBlock,
		TokenStringDouble, TokenStringSingle, TokenVerbatimStringDouble,
		TokenVerbatimStringSingle, TokenNumber:
		return pos, nil
	case TokenOperator:
		if isUnaryOp(m.data(pos)) {
			return m.Expr(pos + 1)
		}
	case TokenAssert:
		if m.kind(pos+1) == TokenSemicolon {
			return m.Expr(pos + 2)
		}
	case TokenBraceL:
		return m.Objinside(pos)
	case TokenBracketL:
		if m.kind(pos+1) == TokenBracketR {
			// empty array
			return pos + 1, nil
		}

		// Test for an expression
		end, err := m.Expr(pos + 1)
		if err != nil {
			return 0, err
		}

		if m.hasTrailingComma(end) {
			end++
		}

		if m.kind(end) == TokenBracketR {
			return end, nil
		}

		if m.kind(end+1) == TokenFor {
			return m.arrayComprehension(end + 1)
		}

		for i := end + 1; i < m.len(); i++ {
			if m.kind(i) == TokenBracketR {
				return i, nil
			}

			i, err = m.Expr(i)
			if err != nil {
				return 0, err
			}

			if m.kind(i+1) == TokenComma {
				if m.kind(i+2) == TokenBracketR {
					return i + 2, nil
				}

				i = i + 2
				continue
			} else if m.kind(i+1) == TokenBracketR {
				return i + 1, nil
			}

			return 0, errors.New("expected ',' after expression")
		}
		return 0, errors.New("array not matched")
	case TokenError:
		end, err := m.Expr(pos + 1)
		if err != nil {
			return 0, err
		}
		return end, nil
	case TokenFunction:
		if m.kind(pos+1) == TokenParenL {
			end, err := m.Params(pos + 2)
			if err != nil {
				return 0, err
			}
			if m.kind(end+1) == TokenParenR {
				return m.Expr(end + 2)
			}
		}
		return 0, errors.New("function not matched")
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
			end, err := m.Params(pos + 2)
			if err != nil {
				return 0, err
			}

			if m.kind(end+1) == TokenParenR {
				return end + 1, nil
			}

			return 0, errors.New("parameters didn't end with a bracket")
		}

		return pos, nil
	case TokenIf:
		end, err := m.Expr(pos + 1)
		if err != nil {
			return 0, err
		}
		if m.kind(end+1) == TokenThen {
			end, err = m.Expr(end + 2)
			if err != nil {
				return 0, err
			}

			if m.kind(end+1) == TokenElse {
				return m.Expr(end + 2)
			}

			return end, nil
		}

		return 0, errors.New("conditional not matched")
	case TokenImport:
		if isString(m.Tokens[pos+1]) {
			return pos + 1, nil
		}
		return 0, errors.New("import not matched")
	case TokenImportStr:
		if isString(m.Tokens[pos+1]) {
			return pos + 1, nil
		}
		return 0, errors.New("importstr not matched")
	case TokenLocal:
		for i := pos + 1; i < len(m.Tokens); i++ {
			m.pos = i
			_, end, err := m.bind()
			if err != nil {
				return 0, err
			}

			if m.kind(end+1) == TokenComma {
				i = end + 1
				continue
			}

			pos = end
			break
		}

		if m.kind(pos+1) == TokenSemicolon {
			return m.Expr(pos + 2)
		}

		return 0, errors.New("expr local not matched")
	case TokenSuper:
		if t := m.Tokens[pos+1]; t.Kind == TokenDot {
			if m.kind(pos+2) == TokenIdentifier {
				return pos + 2, nil
			}
		} else if m.kind(pos+1) == TokenBracketL {
			end, err := m.Expr(pos + 2)
			if err != nil {
				return 0, err
			}

			if m.kind(end+1) == TokenBracketR {
				return end + 1, nil
			}
		}

		return 0, errors.New("super not matched")
	}

	loc := m.loc(pos)
	return 0, errors.Errorf("expr: token %q not matched at %s", m.data(pos), loc.String())
}

func (m *Match) arrayComprehension(pos int) (int, error) {
	end, err := m.Forspec2(pos)
	if err != nil {
		return 0, err
	}

	if m.kind(end+1) == TokenBracketR {
		return end + 1, nil
	}

	return 0, errors.New("expected ']'")

}

// Objinside returns the ending position of an item inside an object.
// nolint: gocyclo
func (m *Match) Objinside(pos int) (int, error) {
	if m.kind(pos) != TokenBraceL {
		return 0, errors.Errorf("expected '{' at %s", m.locString(pos))
	}

	if m.kind(pos+1) == TokenBraceR {
		return pos + 1, nil
	}

	cur := pos

	// If the first token is an object local, this could be a
	// comprehension.
	if m.kind(cur+1) == TokenLocal {
		end, err := m.Objlocal(cur + 1)
		if err == nil {
			cur = end
		}

		if !m.hasTrailingComma(cur) {
			return 0, errors.New("expected ','")
		}
		cur = cur + 1
	}

	// If the current token is a TokenBracketL, this is an object
	// comprehension.
	if m.kind(cur+1) == TokenBracketL {
		end, err := m.Expr(cur + 2)
		if err != nil {
			return 0, err
		}

		cur = end

		if m.kind(cur+1) != TokenBracketR {
			return 0, errors.New("expected ']'")
		}

		cur = cur + 1

		if !m.isOperator(cur+1, ":") {
			return 0, errors.New("expected ':'")
		}

		cur = cur + 1

		end, err = m.Expr(cur + 1)
		if err != nil {
			return 0, err
		}

		cur = end

		if m.hasTrailingComma(cur) {
			cur += 2
			if m.kind(cur) == TokenLocal {
				end, err = m.Objlocal(cur)
				if err != nil {
					return 0, nil
				}
				cur = end

				if m.hasTrailingComma(cur) {
					cur += 2
				}
			}

		} else {
			cur++
		}

		m.pos = cur
		err = m.Forspec()
		if err != nil {
			return 0, err
		}
		end = m.pos

		if m.kind(end+1) != TokenBraceR {
			return 0, errors.New("expected '}'")
		}

		return end + 1, nil
	}

	for i := cur + 1; i < m.len(); i++ {
		end, err := m.Member(i)
		if err != nil {
			return 0, err
		}

		if m.hasTrailingComma(end) {
			end = end + 1
		}

		if m.kind(end+1) == TokenBraceR {
			return end + 1, nil
		}

		i = end
	}

	return 0, errors.New("did not match object inside")
}

// Member returns the ending position of a member started at pos.
func (m *Match) Member(pos int) (int, error) {
	switch m.kind(pos) {
	case TokenLocal:
		return m.Objlocal(pos)
	case TokenAssert:
		return m.Assert(pos)
	case TokenIdentifier, TokenStringDouble, TokenStringSingle:
		return m.Field(pos)
	default:
		return 0, errors.New("did not match object member")
	}
}

// Objlocal returns the ending position of an object local started at pos.
func (m *Match) Objlocal(pos int) (int, error) {
	if m.kind(pos) == TokenLocal {
		m.pos = pos + 1
		_, end, err := m.bind()
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

		if m.isOperator(end+1, ":") {
			msgEnd, err := m.Expr(end + 2)
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

// Field returns the ending position of a field starting at pos.
func (m *Match) Field(pos int) (int, error) {
	end, err := m.Fieldname(pos)
	if err != nil {
		return 0, err
	}

	if m.kind(end+1) == TokenParenL {
		end, err = m.Params(end + 2)
		if err != nil {
			return 0, err
		}

		if m.kind(end+1) != TokenParenR {
			return 0, errors.New("expected ')'")
		}

		end = end + 1
	}

	if m.isFieldVisibility(end + 1) {
		return m.Expr(end + 2)
	}

	return 0, errors.New("did not match a field")
}

func (m *Match) isKind(pos int, kinds []TokenKind) bool {
	posKind := m.kind(pos)

	for _, kind := range kinds {
		if posKind == kind {
			return true
		}
	}

	return false
}

// Params returns the ending position of params starting at pos.
func (m *Match) Params(pos int) (int, error) {
	inOptional := false
	for cur := pos; cur < len(m.Tokens)-1; cur++ {

		// cur, err := m.Expr(cur)
		// if err != nil {
		// 	return 0, err
		// }

		// if (m.kind(cur) != TokenIdentifier) && (!m.isString(cur)) &&
		// 	m.kind(cur) != TokenNumber && m.kind(cur) != TokenSelf {
		// 	loc := m.loc(cur)
		// 	return 0, errors.Errorf("expected an identifier at %d (%s) was %s",
		// 		cur, loc.String(), m.kind(cur))
		// }

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

func (m *Match) ifspec() error {
	if m.kind2(0) == TokenIf {
		end, err := m.Expr(m.pos + 1)
		if err != nil {
			return err
		}

		m.pos = end
		return nil
	}

	return errors.New("did not match ifspec")
}

func (m *Match) Forspec2(pos int) (int, error) {
	m.pos = pos
	if err := m.Forspec(); err != nil {
		return 0, err
	}

	return m.pos, nil
}

func (m *Match) Forspec() error {
	if m.kind2(0) == TokenFor &&
		m.kind2(1) == TokenIdentifier &&
		m.kind2(2) == TokenIn {
		end, err := m.Expr(m.pos + 3)
		if err != nil {
			return err
		}
		m.pos = end
		return nil
	}

	return errors.New("did not match forspec")
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

func (m *Match) loc(pos int) ast.LocationRange {
	return m.Tokens[pos].Loc
}

func (m *Match) locString(pos int) string {
	loc := m.Tokens[pos].Loc
	return loc.String()
}

func (m *Match) kind(pos int) TokenKind {
	return m.Tokens[pos].Kind
}

func (m *Match) kind2(pos int) TokenKind {
	return m.Tokens[m.pos+pos].Kind
}

func (m *Match) data(pos int) string {
	return m.Tokens[pos].Data
}

func (m *Match) data2(pos int) string {
	return m.Tokens[m.pos+pos].Data
}

func (m *Match) isOperator(pos int, name string) bool {
	return m.kind(pos) == TokenOperator &&
		m.data(pos) == name
}

var fieldVisibilities = map[string]ast.ObjectFieldHide{
	":":    ast.ObjectFieldInherit,
	"::":   ast.ObjectFieldHidden,
	":::":  ast.ObjectFieldVisible,
	"+:":   ast.ObjectFieldInherit,
	"+::":  ast.ObjectFieldHidden,
	"+:::": ast.ObjectFieldVisible,
}

func (m *Match) isFieldVisibility(pos int) bool {
	if m.kind(pos) != TokenOperator {
		return false
	}

	_, ok := fieldVisibilities[m.data(pos)]
	return ok
}

func (m *Match) len() int {
	return len(m.Tokens)
}

func (m *Match) incr(i int) {
	m.pos += i
}

func (m *Match) hasTrailingComma(pos int) bool {
	return m.kind(pos+1) == TokenComma
}

func (m *Match) isString(pos int) bool {
	return isString(m.Tokens[pos])
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

func isUnaryOp(op string) bool {
	for k := range ast.UopMap {
		if op == k {
			return true
		}
	}

	return false
}

func isBinaryOp(op string) bool {
	for k := range ast.BopMap {
		if op == k {
			return true
		}
	}

	return false
}

func printTokens(tokens ...Token) {
	for i, t := range tokens {
		fmt.Printf("%d %s: %s = %s\n", i, t.Loc.String(), t.Kind.String(), t.Data)
	}
}

// nolint: gocyclo
func inRange(l ast.Location, lr ast.LocationRange) bool {
	if lr.Begin.Line == l.Line && l.Line == lr.End.Line &&
		lr.Begin.Column <= l.Column && l.Column <= lr.End.Column {
		return true
	} else if lr.Begin.Line < l.Line && l.Line == lr.End.Line &&
		l.Column <= lr.End.Column {
		return true
	} else if lr.Begin.Line == l.Line && l.Line < lr.End.Line &&
		l.Column >= lr.Begin.Column {
		return true
	} else if lr.Begin.Line < l.Line && l.Line < lr.End.Line {
		return true
	}

	return false
}
