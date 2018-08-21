package token

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

type precedence int

const (
	applyPrecedence precedence = 2  // ast.Function calls and indexing.
	unaryPrecedence precedence = 4  // Logical and bitwise negation, unary + -
	maxPrecedence   precedence = 16 // ast.Local, If, ast.Import, ast.Function, Error
)

func Parse(filename, source string) error {
	tokens, err := Lex(filename, source)
	if err != nil {
		return err
	}

	p := mParser{tokens: tokens}
	_, err = p.parse(maxPrecedence)
	return err
}

var bopPrecedence = map[ast.BinaryOp]precedence{
	ast.BopMult:            5,
	ast.BopDiv:             5,
	ast.BopPercent:         5,
	ast.BopPlus:            6,
	ast.BopMinus:           6,
	ast.BopShiftL:          7,
	ast.BopShiftR:          7,
	ast.BopGreater:         8,
	ast.BopGreaterEq:       8,
	ast.BopLess:            8,
	ast.BopLessEq:          8,
	ast.BopIn:              8,
	ast.BopManifestEqual:   9,
	ast.BopManifestUnequal: 9,
	ast.BopBitwiseAnd:      10,
	ast.BopBitwiseXor:      11,
	ast.BopBitwiseOr:       12,
	ast.BopAnd:             13,
	ast.BopOr:              14,
}

// locFromTokenAST creates a location range from a token to a node.
func locFromTokenAST(begin *Token, end ast.Node) ast.LocationRange {
	return ast.LocationRangeBetween(&begin.Loc, end.Loc())
}

type mParser struct {
	tokens Tokens
	cur    int
}

func (p *mParser) parse(prec precedence) (ast.Node, error) {
	begin := p.peek()

	switch begin.Kind {
	case TokenLocal:
		p.pop()
		var binds ast.LocalBinds
		for {
			err := p.parseBind(&binds)
			if err != nil {
				// TODO what should we return here?
			}
			delim := p.pop()
			if delim.Kind != TokenSemicolon && delim.Kind != TokenComma {
				return nil, p.locError(errors.Errorf("expected , or ; but got %v", delim), delim.Loc)
			}
			if delim.Kind == TokenSemicolon {
				break
			}
		}
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		return &ast.Local{
			NodeBase: ast.NewNodeBaseLoc(locFromTokenAST(begin, body)),
			Binds:    binds,
			Body:     body,
		}, nil
	default:
		if begin.Kind == TokenOperator {
		}

	}

	return nil, nil
}

func (p *mParser) parseBind(binds *ast.LocalBinds) error {
	varID, err := p.popExpect(TokenIdentifier)
	if err != nil {
		return err
	}

	for _, b := range *binds {
		if b.Variable == ast.Identifier(varID.Data) {
			return p.locError(errors.Errorf("duplicate local var: %v", varID.Data), varID.Loc)
		}
	}

	var fun *ast.Function
	if p.peek().Kind == TokenParenL {
		p.pop()
		params, gotComma, err := p.parseParameters("function parameter")
		if err != nil {
			return err
		}
		fun = &ast.Function{
			Parameters:    *params,
			TrailingComma: gotComma,
		}
	}

	_, err = p.popExpectOp("=")
	if err != nil {
		return err
	}
	body, err := p.parse(maxPrecedence)
	if err != nil {
		return err
	}

	if fun != nil {
		fun.NodeBase = ast.NewNodeBaseLoc(locFromTokenAST(varID, body))
		fun.Body = body
		*binds = append(*binds, ast.LocalBind{
			Variable: ast.Identifier(varID.Data),
			Body:     body,
			Fun:      fun,
		})
	} else {
		*binds = append(*binds, ast.LocalBind{
			Variable: ast.Identifier(varID.Data),
			Body:     body,
		})
	}

	return nil
}

func (p *mParser) parseArgument() (*ast.Identifier, ast.Node, error) {
	var id *ast.Identifier
	if p.peek().Kind == TokenIdentifier && p.doublePeek().Kind == TokenOperator && p.doublePeek().Data == "=" {
		ident := p.pop()
		var tmpID = ast.Identifier(ident.Data)
		id = &tmpID
		p.pop() // "=" token
	}
	expr, err := p.parse(maxPrecedence)
	if err != nil {
		return nil, nil, err
	}
	return id, expr, nil
}

func (p *mParser) parseArguments(elementKind string) (*Token, *ast.Arguments, bool, error) {
	args := &ast.Arguments{}
	gotComma := false
	namedArgumentAdded := false
	first := true
	for {
		next := p.peek()

		if next.Kind == TokenParenR {
			// gotComma can be true or false here.
			return p.pop(), args, gotComma, nil
		}

		if !first && gotComma {
			return nil, nil, false, p.locError(errors.Errorf("expected a comma before next %s, got %s.", elementKind, next), next.Loc)
		}

		id, expr, err := p.parseArgument()
		if err != nil {
			return nil, nil, false, err
		}
		if id == nil {
			if namedArgumentAdded {
				return nil, nil, false, p.locError(errors.Errorf("positional argument afeter a named argument is not allowed"), next.Loc)
			}
			args.Positional = append(args.Positional, expr)
		} else {
			namedArgumentAdded = true
			args.Named = append(args.Named, ast.NamedArgument{Name: *id, Arg: expr})
		}

		if p.peek().Kind == TokenComma {
			p.pop()
			gotComma = true
		} else {
			gotComma = false
		}

		first = false
	}
}

func (p *mParser) parseParameters(elementKind string) (*ast.Parameters, bool, error) {
	_, args, trailingComma, err := p.parseArguments(elementKind)
	if err != nil {
		return nil, false, err
	}

	var params ast.Parameters
	for _, arg := range args.Positional {
		id, ok := astVarToIdentifier(arg)
		if !ok {
			return nil, false, p.locError(
				errors.Errorf("expected simple identifer but got a complex expression"), *arg.Loc())
		}
		params.Required = append(params.Required, *id)
	}
	for _, arg := range args.Named {
		params.Optional = append(params.Optional, ast.NamedParameter{Name: arg.Name, DefaultArg: arg.Arg})
	}
	return &params, trailingComma, nil

}

func (p *mParser) peek() *Token {
	return &p.tokens[p.cur]
}

func (p *mParser) doublePeek() *Token {
	return &p.tokens[p.cur+1]
}

func (p *mParser) pop() *Token {
	t := p.peek()
	p.cur++
	return t
}

func (p *mParser) popExpect(tk TokenKind) (*Token, error) {
	t := p.pop()
	if t.Kind != tk {
		return nil, p.unexpectedTokenError(tk, t)
	}

	return t, nil
}

func (p *mParser) popExpectOp(op string) (*Token, error) {
	t := p.pop()
	if t.Kind != TokenOperator || t.Data != op {
		return nil, p.locError(
			errors.Errorf("expected operator %v but got %v", op, t), t.Loc)
	}
	return t, nil
}

func (p *mParser) unexpectedTokenError(tk TokenKind, t *Token) error {
	return errors.Errorf("expected token %v but got %v", tk, t)
}

func (p *mParser) locError(err error, loc ast.LocationRange) error {
	return errors.Wrapf(err, "at %s", loc.String())
}

// astVarToIdentifier converts a Var to an Identifier.
//
// in some cases it's convenient to parse something as an expression, and later
// decide that it should be just an identifer
func astVarToIdentifier(node ast.Node) (*ast.Identifier, bool) {
	v, ok := node.(*ast.Var)
	if ok {
		return &v.Id, true
	}
	return nil, false
}
