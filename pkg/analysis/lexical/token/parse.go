package token

import (
	"fmt"
	"strconv"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/parser"
	"github.com/pkg/errors"
)

type precedence int

const (
	applyPrecedence precedence = 2  // ast.Function calls and indexing.
	unaryPrecedence precedence = 4  // Logical and bitwise negation, unary + -
	maxPrecedence   precedence = 16 // ast.Local, If, ast.Import, ast.Function, Error
)

// Parse parses sources into a Jsonnet node.
func Parse(filename, source string, diagnostics chan<- ParseDiagnostic) (ast.Node, error) {
	tokens, err := Lex(filename, source)
	if err != nil {
		return nil, errors.Wrap(err, "lexing source")
	}

	p := mParser{
		tokens: tokens,
		diagCh: diagnostics,
	}

	if diagnostics != nil {
		defer close(diagnostics)
	}

	return p.parse(maxPrecedence)
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
	return ast.LocationRange{
		FileName: begin.Loc.FileName,
		Begin:    begin.Loc.Begin,
		End:      end.Loc().End,
	}
}

// locFromTokens creates a location range from a begin and end token.
func locFromTokens(begin, end *Token) ast.LocationRange {
	return ast.LocationRangeBetween(&begin.Loc, &end.Loc)
}

func locFromPartial(begin *Token) ast.LocationRange {
	return ast.LocationRange{
		FileName: begin.Loc.FileName,
		Begin:    begin.Loc.Begin,
	}
}

// ParseDiagnostic is a diagnostic message about a parse.
type ParseDiagnostic struct {
	Message string
	Loc     ast.LocationRange
}

type mParser struct {
	tokens Tokens
	cur    int
	diagCh chan<- ParseDiagnostic
}

// nolint: gocyclo
func (p *mParser) parse(prec precedence) (ast.Node, error) {
	begin := p.peek()

	switch begin.Kind {
	// These cases have effectively maxPrecedence as the first
	// call to parse will parse them.
	case TokenAssert:
		p.pop()
		cond, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		var msg ast.Node
		if p.peek().Kind == TokenOperator && p.peek().Data == ":" {
			p.pop()
			msg, err = p.parse(maxPrecedence)
			if err != nil {
				return nil, err
			}
		}
		_, err = p.popExpect(TokenSemicolon)
		if err != nil {
			return nil, err
		}
		rest, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		return &ast.Assert{
			NodeBase: ast.NewNodeBaseLoc(locFromTokenAST(begin, rest)),
			Cond:     cond,
			Message:  msg,
			Rest:     rest,
		}, nil

	case TokenIf:
		p.pop()
		cond, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		_, err = p.popExpect(TokenThen)
		if err != nil {
			return nil, err
		}
		branchTrue, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		var branchFalse ast.Node
		lr := locFromTokenAST(begin, branchTrue)
		if p.peek().Kind == TokenElse {
			p.pop()
			branchFalse, err = p.parse(maxPrecedence)
			if err != nil {
				return nil, err
			}
			lr = locFromTokenAST(begin, branchFalse)
		}
		return &ast.Conditional{
			NodeBase:    ast.NewNodeBaseLoc(lr),
			Cond:        cond,
			BranchTrue:  branchTrue,
			BranchFalse: branchFalse,
		}, nil

	case TokenFunction:
		p.pop()
		next := p.pop()
		if next.Kind == TokenParenL {
			params, gotComma, err := p.parseParameters("function parameter")
			if err != nil {
				return nil, err
			}
			body, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, err
			}
			return &ast.Function{
				NodeBase:      ast.NewNodeBaseLoc(locFromTokenAST(begin, body)),
				Parameters:    *params,
				TrailingComma: gotComma,
				Body:          body,
			}, nil
		}
		return nil, locError(errors.Errorf("expected ( but got %v", next), next.Loc)

	case TokenImport:
		p.pop()
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		if lit, ok := body.(*ast.LiteralString); ok {
			if lit.Kind == ast.StringBlock {
				return nil, locError(errors.New("block string literals not allowed in imports"), *body.Loc())
			}
			return &ast.Import{
				NodeBase: ast.NewNodeBaseLoc(locFromTokenAST(begin, body)),
				File:     lit,
			}, nil
		}
		return nil, locError(errors.New("computed imports are not allowed"), *body.Loc())

	case TokenImportStr:
		p.pop()
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		if lit, ok := body.(*ast.LiteralString); ok {
			if lit.Kind == ast.StringBlock {
				return nil, locError(errors.New("block string literals not allowed in imports"), *body.Loc())
			}
			return &ast.ImportStr{
				NodeBase: ast.NewNodeBaseLoc(locFromTokenAST(begin, body)),
				File:     lit,
			}, nil
		}
		return nil, locError(errors.New("Computed imports are not allowed"), *body.Loc())

	case TokenLocal:
		p.pop()

		local := &ast.Local{}

		for {
			err := p.parseBind(&local.Binds)
			if err != nil {
				return nil, err
			}

			if p.atEnd() {
				break
			}

			delim := p.pop()
			if delim.Kind != TokenSemicolon && delim.Kind != TokenComma {
				return nil, locError(errors.Errorf("expected , or ; but got %v", delim), delim.Loc)
			}
			if delim.Kind == TokenSemicolon {
				break
			}
		}

		if max := len(p.tokens) - 1; p.cur > max {
			p.cur = max
		}
		fieldBodyStart := p.tokens[p.cur]

		var body ast.Node
		var err error
		if p.atEnd() {
			p.publishDiag("local assignment is missing", locFromTokens(&fieldBodyStart, p.peek()))
			body = &astext.Partial{
				NodeBase: ast.NewNodeBaseLoc(locFromPartial(p.peekPrev())),
			}
		} else {
			body, err = p.parse(maxPrecedence)
			if err != nil {
				p.publishDiag("local body is not defined", locFromTokens(&fieldBodyStart, p.peek()))
				body = &astext.Partial{
					NodeBase: ast.NewNodeBaseLoc(locFromPartial(p.peekPrev())),
				}
			}
		}

		local.Body = body
		local.NodeBase = ast.NewNodeBaseLoc(locFromTokenAST(begin, body))

		return local, nil

	default:
		// ast.Unary operator
		if begin.Kind == TokenOperator {
			uop, ok := ast.UopMap[begin.Data]
			if !ok {
				return nil, locError(
					errors.Errorf("not a unary operator: %v", begin.Data), begin.Loc)
			}
			if prec == unaryPrecedence {
				op := p.pop()
				expr, err := p.parse(prec)
				if err != nil {
					return nil, err
				}
				return &ast.Unary{
					NodeBase: ast.NewNodeBaseLoc(locFromTokenAST(op, expr)),
					Op:       uop,
					Expr:     expr,
				}, nil
			}
		}

		// Base case
		if prec == 0 {
			return p.parseTerminal()
		}

		lhs, err := p.parse(prec - 1)
		if err != nil {
			return nil, err
		}

		for {
			// The next token must be a binary operator.
			var bop ast.BinaryOp

			// Check precedence is correct for this level. If we're parsing operators
			// with higher precedence, then return lhs and let lower levels deal with
			// the operator.
			switch p.peek().Kind {
			case TokenIn:
				bop = ast.BopIn
				if bopPrecedence[bop] != prec {
					return lhs, nil
				}
			case TokenOperator:
				_ = "breakpoint"
				if p.peek().Data == ":" {
					// Special case for the colons in assert. Since COLON is no longer a
					// special token, we have to make sure it does not trip the
					// op_is_binary test below. It should terminal parsing of the
					// expression here, returning control to the parsing of the actual
					// assert AST.
					return lhs, nil
				}
				var ok bool
				bop, ok = ast.BopMap[p.peek().Data]
				if !ok {
					return nil, locError(errors.Errorf("not a binary operator: %v", p.peek().Data), p.peek().Loc)
				}

				if bopPrecedence[bop] != prec {
					return lhs, nil
				}

			case TokenDot, TokenBracketL, TokenParenL, TokenBraceL:
				if applyPrecedence != prec {
					return lhs, nil
				}
			default:
				return lhs, nil
			}

			op := p.pop()
			switch op.Kind {
			case TokenBracketL:
				// handle slice
				var indexes [3]ast.Node
				colonsConsumed := 0

				var end *Token
				readyForNextIndex := true
				for colonsConsumed < 3 {
					if p.peek().Kind == TokenBracketR {
						end = p.pop()
						break
					} else if p.peek().Data == ":" {
						colonsConsumed++
						end = p.pop()
						readyForNextIndex = true
					} else if p.peek().Data == "::" {
						colonsConsumed += 2
						end = p.pop()
						readyForNextIndex = true
					} else if readyForNextIndex {
						indexes[colonsConsumed], err = p.parse(maxPrecedence)
						if err != nil {
							return nil, err
						}
						readyForNextIndex = false
					} else {
						return nil, p.unexpectedTokenError(TokenBracketR, p.peek())
					}
				}

				if colonsConsumed > 2 {
					// example: target[42:42:42:42]
					return p.parsingFailure("invalid slice: too many colons", end)
				}
				if colonsConsumed == 0 && readyForNextIndex {
					// example: target[]
					return p.parsingFailure("ast.Index requires an expression", end)
				}
				isSlice := colonsConsumed > 0

				if isSlice {
					lhs = &ast.Slice{
						NodeBase:   ast.NewNodeBaseLoc(locFromTokens(begin, end)),
						Target:     lhs,
						BeginIndex: indexes[0],
						EndIndex:   indexes[1],
						Step:       indexes[2],
					}
				} else {
					lhs = &ast.Index{
						NodeBase: ast.NewNodeBaseLoc(locFromTokens(begin, end)),
						Target:   lhs,
						Index:    indexes[0],
					}
				}

			case TokenDot:
				fieldID, err := p.popExpect(TokenIdentifier)
				if err != nil {
					return nil, err
				}
				id := ast.Identifier(fieldID.Data)
				lhs = &ast.Index{
					NodeBase: ast.NewNodeBaseLoc(locFromTokens(begin, fieldID)),
					Target:   lhs,
					Id:       &id,
				}

			case TokenParenL:
				end, args, gotComma, err := p.parseArguments("function argument")
				if err != nil {
					return nil, err
				}
				tailStrict := false
				if p.peek().Kind == TokenTailStrict {
					p.pop()
					tailStrict = true
				}
				lhs = &ast.Apply{
					NodeBase:      ast.NewNodeBaseLoc(locFromTokens(begin, end)),
					Target:        lhs,
					Arguments:     *args,
					TrailingComma: gotComma,
					TailStrict:    tailStrict,
				}

			case TokenBraceL:
				obj, end, err := p.parseObjectRemainder(op)
				if err != nil {
					return nil, err
				}
				lhs = &ast.ApplyBrace{
					NodeBase: ast.NewNodeBaseLoc(locFromTokens(begin, end)),
					Left:     lhs,
					Right:    obj,
				}
			default:
				if op.Kind == TokenIn && p.peek().Kind == TokenSuper {
					super := p.pop()
					lhs = &ast.InSuper{
						NodeBase: ast.NewNodeBaseLoc(locFromTokens(begin, super)),
						Index:    lhs,
					}
				} else {
					rhs, err := p.parse(prec - 1)
					if err != nil {
						return nil, err
					}
					lhs = &ast.Binary{
						NodeBase: ast.NewNodeBaseLoc(locFromTokenAST(begin, rhs)),
						Left:     lhs,
						Op:       bop,
						Right:    rhs,
					}
				}
			}
		}
	}
}

func (p *mParser) parseBind(binds *ast.LocalBinds) error {
	varID, err := p.popExpect(TokenIdentifier)
	if err != nil {
		return err
	}

	for _, b := range *binds {
		if b.Variable == ast.Identifier(varID.Data) {
			return locError(errors.Errorf("duplicate local var: %v", varID.Data), varID.Loc)
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
		// body could be invalid in a completion event
		p.publishDiag("bind body is incomplete", locFromPartial(p.peekPrev()))
		body = &astext.Partial{
			NodeBase: ast.NewNodeBaseLoc(locFromPartial(p.peekPrev())),
		}
	}

	loc := locFromTokens(varID, varID)
	if fun != nil {
		fun.NodeBase = ast.NewNodeBaseLoc(locFromTokenAST(varID, body))
		fun.Body = body
		*binds = append(*binds, ast.LocalBind{
			Variable: ast.Identifier(varID.Data),
			Body:     body,
			Fun:      fun,
			VarLoc:   loc,
		})
	} else {
		*binds = append(*binds, ast.LocalBind{
			Variable: ast.Identifier(varID.Data),
			Body:     body,
			VarLoc:   loc,
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

		if !first && !gotComma {
			return nil, nil, false, parser.MakeStaticError(fmt.Sprintf("Expected a comma before next %s, got %s.", elementKind, next), next.Loc)
		}

		id, expr, err := p.parseArgument()
		if err != nil {
			return nil, nil, false, err
		}
		if id == nil {
			if namedArgumentAdded {
				return nil, nil, false, parser.MakeStaticError("Positional argument after a named argument is not allowed", next.Loc)
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

// parseArray parses arrays.
//
// Assumes that the leading '[' has already been consumed and passed as tok.
// Should read up to and consume the trailing ']'
// nolint: gocyclo
func (p *mParser) parseArray(tok *Token) (ast.Node, error) {
	next := p.peek()
	if next.Kind == TokenBracketR {
		p.pop()
		return &ast.Array{
			NodeBase: ast.NewNodeBaseLoc(locFromTokens(tok, next)),
		}, nil
	}

	first, err := p.parse(maxPrecedence)
	if err != nil {
		return nil, err
	}
	var gotComma bool
	next = p.peek()
	if next.Kind == TokenComma {
		p.pop()
		next = p.peek()
		gotComma = true
	}

	if next.Kind == TokenFor {
		// It's a comprehension
		p.pop()
		spec, last, err := p.parseComprehensionSpecs(TokenBracketR)
		if err != nil {
			return nil, err
		}
		return &ast.ArrayComp{
			NodeBase:      ast.NewNodeBaseLoc(locFromTokens(tok, last)),
			Body:          first,
			TrailingComma: gotComma,
			Spec:          *spec,
		}, nil
	}
	// Not a comprehension: It can have more elements.
	elements := ast.Nodes{first}

	for {
		if next.Kind == TokenBracketR {
			// TODO(dcunnin): SYNTAX SUGAR HERE (preserve comma)
			p.pop()
			break
		}
		if !gotComma {
			return nil, locError(errors.New("expected a comma before next array element"), next.Loc)
		}
		nextElem, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		elements = append(elements, nextElem)
		next = p.peek()
		if next.Kind == TokenComma {
			p.pop()
			next = p.peek()
			gotComma = true
		} else {
			gotComma = false
		}
	}

	return &ast.Array{
		NodeBase:      ast.NewNodeBaseLoc(locFromTokens(tok, next)),
		Elements:      elements,
		TrailingComma: gotComma,
	}, nil
}

/* parseComprehensionSpecs parses for x in expr for y in expr if expr for z in expr ... */
func (p *mParser) parseComprehensionSpecs(end TokenKind) (*ast.ForSpec, *Token, error) {
	var parseComprehensionSpecsHelper func(outer *ast.ForSpec) (*ast.ForSpec, *Token, error)
	parseComprehensionSpecsHelper = func(outer *ast.ForSpec) (*ast.ForSpec, *Token, error) {
		var ifSpecs []ast.IfSpec

		varID, err := p.popExpect(TokenIdentifier)
		if err != nil {
			return nil, nil, err
		}
		id := ast.Identifier(varID.Data)
		_, err = p.popExpect(TokenIn)
		if err != nil {
			return nil, nil, err
		}
		arr, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, nil, err
		}
		forSpec := &ast.ForSpec{
			VarName: id,
			Expr:    arr,
			Outer:   outer,
		}

		maybeIf := p.pop()
		for ; maybeIf.Kind == TokenIf; maybeIf = p.pop() {
			cond, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, nil, err
			}
			ifSpecs = append(ifSpecs, ast.IfSpec{
				Expr: cond,
			})
		}
		forSpec.Conditions = ifSpecs
		if maybeIf.Kind == end {
			return forSpec, maybeIf, nil
		}

		if maybeIf.Kind != TokenFor {
			return nil, nil, locError(
				errors.Errorf("expected for, if or %v after for clause, got: %v", end, maybeIf), maybeIf.Loc)
		}

		return parseComprehensionSpecsHelper(forSpec)
	}
	return parseComprehensionSpecsHelper(nil)
}

func (p *mParser) parseObjectAssignmentOp() (plusSugar bool, hide ast.ObjectFieldHide, err error) {
	op, err := p.popExpect(TokenOperator)
	if err != nil {
		return
	}
	opStr := op.Data
	if opStr[0] == '+' {
		plusSugar = true
		opStr = opStr[1:]
	}

	numColons := 0
	for len(opStr) > 0 {
		if opStr[0] != ':' {
			err = locError(
				errors.Errorf("expected one of :, ::, :::, +:, +::, +:::, got: %v", op.Data), op.Loc)
			return
		}
		opStr = opStr[1:]
		numColons++
	}

	switch numColons {
	case 1:
		hide = ast.ObjectFieldInherit
	case 2:
		hide = ast.ObjectFieldHidden
	case 3:
		hide = ast.ObjectFieldVisible
	default:
		err = locError(
			errors.Errorf("expected one of :, ::, :::, +:, +::, +:::, got: %v", op.Data), op.Loc)
		return
	}

	return
}

// A LiteralField is a field of an object or object comprehension.
// +gen set
type LiteralField string

// parseObjectRemainder parses object or object comprehension without leading brace
// nolint: gocyclo
func (p *mParser) parseObjectRemainder(tok *Token) (ast.Node, *Token, error) {
	var fields ast.ObjectFields
	literalFields := make(LiteralFieldSet)
	binds := make(ast.IdentifierSet)

	gotComma := false
	first := true

	for {
		next := p.pop()
		if !gotComma && !first {
			if next.Kind == TokenComma {
				next = p.pop()
				gotComma = true
			}
		}

		if next.Kind == TokenBraceR {
			// empty object {}
			return &ast.Object{
				NodeBase:      ast.NewNodeBaseLoc(locFromTokens(tok, next)),
				Fields:        fields,
				TrailingComma: gotComma,
			}, next, nil
		}

		if next.Kind == TokenFor {
			// It's a comprehension
			numFields := 0
			numAsserts := 0
			var field ast.ObjectField
			for _, f := range fields {
				if f.Kind == ast.ObjectLocal {
					continue
				}
				if f.Kind == ast.ObjectAssert {
					numAsserts++
					continue
				}
				numFields++
				field = f
			}

			if numAsserts > 0 {
				return nil, nil, locError(errors.New("object comprehension cannot have asserts"), next.Loc)
			}
			if numFields != 1 {
				return nil, nil, locError(errors.New("object comprehension can only have one field"), next.Loc)
			}
			if field.Hide != ast.ObjectFieldInherit {
				return nil, nil, locError(errors.New("object comprehensions cannot have hidden fields"), next.Loc)
			}
			if field.Kind != ast.ObjectFieldExpr {
				return nil, nil, locError(errors.New("object comprehensions can only have [e] fields"), next.Loc)
			}
			spec, last, err := p.parseComprehensionSpecs(TokenBraceR)
			if err != nil {
				return nil, nil, err
			}
			return &ast.ObjectComp{
				NodeBase:      ast.NewNodeBaseLoc(locFromTokens(tok, last)),
				Fields:        fields,
				TrailingComma: gotComma,
				Spec:          *spec,
			}, last, nil
		}

		if !gotComma && !first {
			return nil, nil, locError(errors.New("expected a comma before next field"), next.Loc)
		}
		first = false

		switch next.Kind {
		case TokenBracketL, TokenIdentifier, TokenStringDouble, TokenStringSingle,
			TokenStringBlock, TokenVerbatimStringDouble, TokenVerbatimStringSingle:
			var kind ast.ObjectFieldKind
			var expr1 ast.Node
			var id *ast.Identifier
			switch next.Kind {
			case TokenIdentifier:
				kind = ast.ObjectFieldID
				id = (*ast.Identifier)(&next.Data)
			case TokenStringDouble, TokenStringSingle,
				TokenStringBlock, TokenVerbatimStringDouble, TokenVerbatimStringSingle:
				kind = ast.ObjectFieldStr
				expr1 = tokenStringToAst(next)
			default:
				kind = ast.ObjectFieldExpr
				var err error
				expr1, err = p.parse(maxPrecedence)
				if err != nil {
					return nil, nil, err
				}
				_, err = p.popExpect(TokenBracketR)
				if err != nil {
					return nil, nil, err
				}
			}

			isMethod := false
			methComma := false
			var params *ast.Parameters
			if p.peek().Kind == TokenParenL {
				p.pop()
				var err error
				params, methComma, err = p.parseParameters("method parameter")
				if err != nil {
					return nil, nil, err
				}
				isMethod = true
			}

			plusSugar, hide, err := p.parseObjectAssignmentOp()
			if err != nil {
				return nil, nil, err
			}

			if plusSugar && isMethod {
				return nil, nil, locError(
					errors.Errorf("Cannot use +: syntax sugar in a method: %v", next.Data), next.Loc)
			}

			if kind != ast.ObjectFieldExpr {
				if !literalFields.Add(LiteralField(next.Data)) {
					return nil, nil, locError(
						errors.Errorf("Duplicate field: %v", next.Data), next.Loc)
				}
			}

			body, err := p.parse(maxPrecedence)
			if err != nil {
				next = p.peek()
				if next.Kind != TokenComma && next.Kind != TokenSemicolon {
					return nil, nil, err

				}
				p.cur = p.cur - 1
				p.publishDiag("object body is incomplete or missing",
					locFromPartial(p.peek()))
				body = &astext.Partial{
					NodeBase: ast.NewNodeBaseLoc(locFromPartial(p.peek())),
				}
			}

			var method *ast.Function
			if isMethod {
				method = &ast.Function{
					Parameters:    *params,
					TrailingComma: methComma,
					Body:          body,
				}
			}

			fields = append(fields, ast.ObjectField{
				Kind:          kind,
				Hide:          hide,
				SuperSugar:    plusSugar,
				MethodSugar:   isMethod,
				Method:        method,
				Expr1:         expr1,
				Id:            id,
				Params:        params,
				TrailingComma: methComma,
				Expr2:         body,
			})

		case TokenLocal:
			varID, err := p.popExpect(TokenIdentifier)
			if err != nil {
				return nil, nil, err
			}

			id := ast.Identifier(varID.Data)

			if binds.Contains(id) {
				return nil, nil, locError(errors.Errorf("duplicate local var: %v", id), varID.Loc)
			}

			// TODO(sbarzowski) Can we reuse regular local bind parsing here?

			isMethod := false
			funcComma := false
			var params *ast.Parameters
			if p.peek().Kind == TokenParenL {
				p.pop()
				isMethod = true
				params, funcComma, err = p.parseParameters("function parameter")
				if err != nil {
					return nil, nil, err
				}
			}
			_, err = p.popExpectOp("=")
			if err != nil {
				return nil, nil, err
			}

			body, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, nil, err
			}

			var method *ast.Function
			if isMethod {
				method = &ast.Function{
					Parameters:    *params,
					TrailingComma: funcComma,
					Body:          body,
				}
			}

			binds.Add(id)

			fields = append(fields, ast.ObjectField{
				Kind:          ast.ObjectLocal,
				Hide:          ast.ObjectFieldVisible,
				SuperSugar:    false,
				MethodSugar:   isMethod,
				Method:        method,
				Id:            &id,
				Params:        params,
				TrailingComma: funcComma,
				Expr2:         body,
			})

		case TokenAssert:
			cond, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, nil, err
			}
			var msg ast.Node
			if p.peek().Kind == TokenOperator && p.peek().Data == ":" {
				p.pop()
				msg, err = p.parse(maxPrecedence)
				if err != nil {
					return nil, nil, err
				}
			}

			fields = append(fields, ast.ObjectField{
				Kind:  ast.ObjectAssert,
				Hide:  ast.ObjectFieldVisible,
				Expr2: cond,
				Expr3: msg,
			})
		default:
			return nil, nil, p.unexpectedError(next, "parsing field definition")
		}
		gotComma = false
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
			return nil, false, locError(
				errors.Errorf("expected simple identifer but got a complex expression"), *arg.Loc())
		}
		params.Required = append(params.Required, *id)
	}
	for _, arg := range args.Named {
		params.Optional = append(params.Optional, ast.NamedParameter{Name: arg.Name, DefaultArg: arg.Arg})
	}
	return &params, trailingComma, nil

}

// nolint: gocyclo
func (p *mParser) parseTerminal() (ast.Node, error) {
	tok := p.pop()
	switch tok.Kind {
	case TokenAssert, TokenBraceR, TokenBracketR, TokenComma, TokenDot, TokenElse,
		TokenError, TokenFor, TokenFunction, TokenIf, TokenIn, TokenImport, TokenImportStr,
		TokenLocal, TokenOperator, TokenParenR, TokenSemicolon, TokenTailStrict, TokenThen:
		return nil, p.unexpectedError(tok, "parsing terminal")

	case TokenEndOfFile:
		return nil, locError(errors.New("unexpected end of file"), tok.Loc)

	case TokenBraceL:
		obj, _, err := p.parseObjectRemainder(tok)
		return obj, err

	case TokenBracketL:
		return p.parseArray(tok)

	case TokenParenL:
		inner, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		tokRight, err := p.popExpect(TokenParenR)
		if err != nil {
			return nil, err
		}
		return &ast.Parens{
			NodeBase: ast.NewNodeBaseLoc(locFromTokens(tok, tokRight)),
			Inner:    inner,
		}, nil

	// Literals
	case TokenNumber:
		// This shouldn't fail as the lexer should make sure we have good input but
		// we handle the error regardless.
		num, err := strconv.ParseFloat(tok.Data, 64)
		if err != nil {
			return nil, locError(errors.New("could not parse floating point number"), tok.Loc)
		}
		return &ast.LiteralNumber{
			NodeBase:       ast.NewNodeBaseLoc(tok.Loc),
			Value:          num,
			OriginalString: tok.Data,
		}, nil
	case TokenStringDouble, TokenStringSingle,
		TokenStringBlock, TokenVerbatimStringDouble, TokenVerbatimStringSingle:
		return tokenStringToAst(tok), nil
	case TokenFalse:
		return &ast.LiteralBoolean{
			NodeBase: ast.NewNodeBaseLoc(tok.Loc),
			Value:    false,
		}, nil
	case TokenTrue:
		return &ast.LiteralBoolean{
			NodeBase: ast.NewNodeBaseLoc(tok.Loc),
			Value:    true,
		}, nil
	case TokenNullLit:
		return &ast.LiteralNull{
			NodeBase: ast.NewNodeBaseLoc(tok.Loc),
		}, nil

	// Variables
	case TokenDollar:
		return &ast.Dollar{
			NodeBase: ast.NewNodeBaseLoc(tok.Loc),
		}, nil
	case TokenIdentifier:
		return &ast.Var{
			NodeBase: ast.NewNodeBaseLoc(tok.Loc),
			Id:       ast.Identifier(tok.Data),
		}, nil
	case TokenSelf:
		return &ast.Self{
			NodeBase: ast.NewNodeBaseLoc(tok.Loc),
		}, nil
	case TokenSuper:
		next := p.pop()
		var index ast.Node
		var id *ast.Identifier
		switch next.Kind {
		case TokenDot:
			fieldID, err := p.popExpect(TokenIdentifier)
			if err != nil {
				return nil, err
			}
			id = (*ast.Identifier)(&fieldID.Data)
		case TokenBracketL:
			var err error
			index, err = p.parse(maxPrecedence)
			if err != nil {
				return nil, err
			}
			_, err = p.popExpect(TokenBracketR)
			if err != nil {
				return nil, err
			}
		default:
			return nil, locError(errors.New("expected . or [ after super"), tok.Loc)
		}
		return &ast.SuperIndex{
			NodeBase: ast.NewNodeBaseLoc(tok.Loc),
			Index:    index,
			Id:       id,
		}, nil
	}

	return nil, locError(errors.Errorf("INTERNAL ERROR: Unknown tok kind: %v", tok.Kind), tok.Loc)
}

func (p *mParser) parsingFailure(msg string, tok *Token) (ast.Node, error) {
	return nil, locError(errors.New(msg), tok.Loc)
}

func (p *mParser) atEnd() bool {
	return p.cur == len(p.tokens)
}

func (p *mParser) loc() string {
	if p.cur >= len(p.tokens)-1 {
		return "eof"
	}

	return p.peek().Loc.String()
}

func (p *mParser) peek() *Token {
	if max := len(p.tokens) - 1; p.cur > max {
		return &p.tokens[max]
	}

	return &p.tokens[p.cur]
}

func (p *mParser) peekPrev() *Token {
	return &p.tokens[p.cur-1]
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
		return nil, locError(
			errors.Errorf("expected operator %v but got %v", op, t), t.Loc)
	}
	return t, nil
}

func (p *mParser) publishDiag(msg string, loc ast.LocationRange) {
	if p.diagCh != nil {
		p.diagCh <- ParseDiagnostic{
			Message: msg,
			Loc:     loc,
		}
	}
}

func (p *mParser) unexpectedError(t *Token, while string) error {
	return locError(errors.Errorf("unexpected: %v while %v", t, while), t.Loc)
}

func (p *mParser) unexpectedTokenError(tk TokenKind, t *Token) error {
	return errors.Errorf("expected token %v but got %v", tk, t)
}

func locError(err error, loc ast.LocationRange) error {
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

func tokenStringToAst(tok *Token) *ast.LiteralString {
	switch tok.Kind {
	case TokenStringSingle:
		return &ast.LiteralString{
			NodeBase: ast.NewNodeBaseLoc(tok.Loc),
			Value:    tok.Data,
			Kind:     ast.StringSingle,
		}
	case TokenStringDouble:
		return &ast.LiteralString{
			NodeBase: ast.NewNodeBaseLoc(tok.Loc),
			Value:    tok.Data,
			Kind:     ast.StringDouble,
		}
	case TokenStringBlock:
		return &ast.LiteralString{
			NodeBase:    ast.NewNodeBaseLoc(tok.Loc),
			Value:       tok.Data,
			Kind:        ast.StringBlock,
			BlockIndent: tok.StringBlockIndent,
		}
	case TokenVerbatimStringDouble:
		return &ast.LiteralString{
			NodeBase: ast.NewNodeBaseLoc(tok.Loc),
			Value:    tok.Data,
			Kind:     ast.VerbatimStringDouble,
		}
	case TokenVerbatimStringSingle:
		return &ast.LiteralString{
			NodeBase: ast.NewNodeBaseLoc(tok.Loc),
			Value:    tok.Data,
			Kind:     ast.VerbatimStringSingle,
		}
	default:
		panic(fmt.Sprintf("Not a string token %#+v", tok))
	}
}
