package fylin

import (
	"errors"
	"fmt"
	"strconv"
)

type defCtx struct {
	encl *defCtx
	*loopCtx
}

type loopCtx struct {
	encl *loopCtx
}

type parser struct {
	scanner  scanner
	current  token
	previous token
	errors   []string
	*defCtx
}

func newParser(source []byte) *parser {
	return &parser{
		scanner: newScanner(source),
		errors:  make([]string, 0),
		defCtx:  nil,
	}
}

func (p *parser) advance() {
	p.previous = p.current
	p.current = p.scanner.scanToken()
}

func (p *parser) block() []astStmt {
	stmts := []astStmt{}
	p.consume(tokenColon, "expected ':'")
	p.consume(tokenNewLine, "expected new line")
	p.consume(tokenIntab, "expected indent")
	for {
		stmts = append(stmts, p.stmt())
		if p.check(tokenDetab) || p.check(tokenEof) {
			break
		}
	}
	p.consume(tokenDetab, "expected dedent")
	return stmts
}

func (p *parser) check(type_ tokenType) bool {
	return p.current.tokenType == type_
}

func (p *parser) match(type_ tokenType) bool {
	if p.check(type_) {
		p.advance()
		return true
	}
	return false
}

func (p *parser) consume(type_ tokenType, message string) {
	if p.match(type_) {
		return
	}
	p.errorAtCurrent(message)
}

type parseError string

func (p *parser) errorAt(tk token, message string) {
	if tk.tokenType == tokenIdentifier ||
		tk.tokenType == tokenString ||
		tk.tokenType == tokenFloat ||
		tk.tokenType == tokenInteger {
		message = fmt.Sprintf("line %d at '%s': %s", tk.line, tk.literal, message)
	} else {
		message = fmt.Sprintf("line %d at token (%s): %s", tk.line, tk.tokenType, message)
	}
	p.errors = append(p.errors, message)
	panic(parseError(message))
}

func (p *parser) errorAtPrevious(message string) {
	p.errorAt(p.previous, message)
}

func (p *parser) errorAtCurrent(message string) {
	p.errorAt(p.current, message)
}

func (p *parser) parse() ([]astStmt, error) {
	program := []astStmt{}
	p.advance()

	for !p.match(tokenEof) {
		program = append(program, p.stmt())
	}

	if len(p.errors) != 0 {
		return nil, errors.New(p.errors[0])
	}
	return program, nil
}

func (p *parser) stmt() (stmt astStmt) {
	defer catch(func(pe parseError) {
		stmt = badStmt(pe)
		p.synchronize()
	})

	if p.match(tokenWhile) {
		return p.whileStmt()
	} else if p.match(tokenLocal) ||
		p.match(tokenNonLocal) ||
		p.match(tokenGlobal) {
		return p.declStmt()
	} else if p.match(tokenRaise) {
		return p.raiseStmt()
	} else if p.match(tokenTry) {
		return p.tryStmt()
	} else if p.match(tokenFor) {
		return p.forStmt()
	} else if p.match(tokenDef) {
		return p.defStmt()
	} else if p.match(tokenIf) {
		return p.ifStmt()
	} else if p.match(tokenBreak) {
		return p.breakStmt()
	} else if p.match(tokenContinue) {
		return p.continueStmt()
	} else if p.match(tokenReturn) {
		return p.returnStmt()
	} else {
		expr := p.expr(precLowest)
		if p.check(tokenEqual) || p.check(tokenComma) {
			return p.assignStmt(expr)
		}
		p.consume(tokenNewLine, "expect new line")
		return &exprStmt{expr}
	}
}

func isLeftHand(expr astExpr) bool {
	switch expr.(type) {
	case *ident, *indexExpr:
		return true
	default:
		return false
	}
}

func (p *parser) declStmt() *declStmt {
	var t varType
	switch p.previous.tokenType {
	case tokenNonLocal:
		t = varNonLocal
	case tokenLocal:
		t = varLocal
	case tokenGlobal:
		t = varGlobal
	}
	decl := &declStmt{varType: t}
	if p.defCtx == nil {
		p.errorAtPrevious("variable modifier outside function")
	}
	for {
		p.consume(tokenIdentifier, "expect variable name")
		decl.vars = append(decl.vars, p.previous.literal)
		if !p.match(tokenComma) {
			break
		}
	}
	p.consume(tokenNewLine, "expect new line")
	return decl
}

func (p *parser) breakStmt() *breakStmt {
	if p.loopCtx == nil {
		p.errorAtPrevious("'break' outside function")
	}
	stmt := &breakStmt{}
	p.consume(tokenNewLine, "expect new line")
	return stmt
}

func (p *parser) continueStmt() *continueStmt {
	if p.loopCtx == nil {
		p.errorAtPrevious("'continue' outside function")
	}
	stmt := &continueStmt{}
	p.consume(tokenNewLine, "expect new line")
	return stmt
}

func (p *parser) raiseStmt() *raiseStmt {
	stmt := &raiseStmt{p.expr(precLowest)}
	p.consume(tokenNewLine, "expect new line")
	return stmt
}

func (p *parser) tryStmt() *tryStmt {
	stmt := &tryStmt{}
	stmt.try = p.block()
	if p.match(tokenExcept) {
		if p.match(tokenAs) {
			p.consume(tokenIdentifier, "expect exception name")
			stmt.as = p.previous.literal
		}
		stmt.except = p.block()
	}
	if p.match(tokenFinally) {
		stmt.finally = p.block()
	}
	if stmt.except == nil && stmt.finally == nil {
		p.errorAtCurrent("expect 'except' or 'finally'")
	}
	return stmt
}

func (p *parser) returnStmt() *returnStmt {
	if p.defCtx == nil {
		p.errorAtPrevious("'return' outside function")
	}
	stmt := &returnStmt{
		values: []astExpr{},
	}
	if p.match(tokenNewLine) {
		return stmt
	}
	for {
		stmt.values = append(stmt.values, p.expr(precLowest))
		if !p.match(tokenComma) {
			break
		}
	}
	p.consume(tokenNewLine, "expect new line")
	return stmt
}

func (p *parser) assignStmt(first astExpr) *assignStmt {
	if !isLeftHand(first) {
		p.errorAtPrevious("wrong assign target")
	}
	stmt := &assignStmt{
		lefts: []astExpr{first},
	}
	for p.match(tokenComma) {
		left := p.expr(precLowest)
		if !isLeftHand(left) {
			p.errorAtPrevious("wrong assign target")
		}
		stmt.lefts = append(stmt.lefts, left)
	}
	p.consume(tokenEqual, "expect '='")
	for {
		stmt.rights = append(stmt.rights, p.expr(precLowest))
		if !p.match(tokenComma) {
			break
		}
	}
	p.consume(tokenNewLine, "expect new line")
	return stmt
}

func (p *parser) expr(prec precedence) astExpr {
	var left astExpr
	p.advance()
	switch p.previous.tokenType {
	case tokenNone:
		left = &noneLit{}
	case tokenFalse:
		left = &boolLit{false}
	case tokenTrue:
		left = &boolLit{true}
	case tokenFloat:
		n, _ := strconv.ParseFloat(p.previous.literal, 64)
		left = &numLit{n}
	case tokenInteger:
		base := integerBases[lowerChar(p.previous.literal[1])]
		n, _ := strconv.ParseUint(p.previous.literal[2:], base, 64)
		left = &numLit{float64(n)}
	case tokenString:
		left = &strLit{p.previous.literal[1 : len(p.previous.literal)-1]}
	case tokenIdentifier:
		left = &ident{p.previous.literal}
	case tokenMinus, tokenPlus, tokenNot:
		left = p.prefixExpr()
	case tokenLeftParen:
		left = p.expr(precLowest)
		p.consume(tokenRightParen, "expect ')'")
	case tokenLeftBracket:
		left = p.listLit()
	case tokenLeftBrace:
		left = p.dictLit()
	default:
		p.errorAtPrevious("expect expression")
	}

	for prec < precedences[p.current.tokenType] {
		p.advance()
		switch p.previous.tokenType {
		case tokenAnd, tokenOr, tokenEqualEqual, tokenBangEqual,
			tokenPlus, tokenMinus, tokenStar, tokenSlash,
			tokenGreater, tokenGreaterEqual,
			tokenLess, tokenLessEqual:
			left = p.infixExpr(left)
		case tokenDot:
			left = p.propertyExpr(left)
		case tokenLeftBracket:
			left = p.indexExpr(left)
		case tokenLeftParen:
			left = p.callExpr(left)
		case tokenLeftBrace:
			left = p.protoDictExpr(left)
		default:
			panic("expr: what?")
		}
	}

	return left
}

func (p *parser) propertyExpr(left astExpr) *indexExpr {
	expr := &indexExpr{
		left: left,
	}
	p.consume(tokenIdentifier, "expect property")
	expr.index = &strLit{p.previous.literal}
	return expr
}

func (p *parser) indexExpr(left astExpr) *indexExpr {
	expr := &indexExpr{
		left: left,
	}
	expr.index = p.expr(precLowest)
	p.consume(tokenRightBracket, "expect ']'")
	return expr
}

func (p *parser) callExpr(left astExpr) *callExpr {
	expr := &callExpr{
		left: left,
	}
	expr.args = p.args()
	return expr
}

func (p *parser) infixExpr(left astExpr) *infixExpr {
	expr := &infixExpr{
		left:    left,
		opToken: p.previous,
	}
	expr.right = p.expr(precedences[p.previous.tokenType])
	return expr
}

func (p *parser) prefixExpr() *prefixExpr {
	expr := &prefixExpr{
		opToken: p.previous,
	}
	expr.right = p.expr(precUnary)
	return expr
}

func (p *parser) protoDictExpr(left astExpr) *protoDictExpr {
	return &protoDictExpr{left, p.dictLit()}
}

func (p *parser) dictLit() *dictLit {
	lit := &dictLit{map[astExpr]astExpr{}}
	if p.match(tokenRightBrace) {
		return lit
	}
	for {
		key := p.expr(precLowest)
		p.consume(tokenColon, "expect ':'")
		val := p.expr(precLowest)
		lit.pairs[key] = val
		if !p.match(tokenComma) {
			break
		}
		if p.check(tokenRightBrace) {
			break
		}
	}
	p.consume(tokenRightBrace, "expect '}'")
	return lit
}

func (p *parser) listLit() *listLit {
	lit := &listLit{[]astExpr{}}
	if p.match(tokenRightBracket) {
		return lit
	}
	for {
		lit.elems = append(lit.elems, p.expr(precLowest))
		if !p.match(tokenComma) {
			break
		}
		if p.check(tokenRightBracket) {
			break
		}
	}
	p.consume(tokenRightBracket, "expect ']'")
	return lit
}

type precedence int

const (
	precLowest precedence = iota
	precOr                // or
	precAnd               // and
	precEq                // == !=
	precComp              // < > <= >=
	precTerm              // + -
	precFact              // * /
	precUnary             // not - +
	precCall              // . () {} []
	precHighest
)

var precedences = map[tokenType]precedence{
	tokenOr: precOr,

	tokenAnd: precAnd,

	tokenEqualEqual: precEq,
	tokenBangEqual:  precEq,

	tokenGreater:      precComp,
	tokenGreaterEqual: precComp,
	tokenLess:         precComp,
	tokenLessEqual:    precComp,

	tokenPlus:  precTerm,
	tokenMinus: precTerm,

	tokenStar:  precFact,
	tokenSlash: precFact,

	tokenDot:         precCall,
	tokenLeftParen:   precCall,
	tokenLeftBracket: precCall,
	tokenLeftBrace:   precCall,
}

func (p *parser) whileStmt() *whileStmt {
	stmt := &whileStmt{}
	stmt.cond = p.expr(precLowest)
	p.loopCtx = &loopCtx{p.loopCtx}
	stmt.loop = p.block()
	p.loopCtx = p.loopCtx.encl
	return stmt
}

func (p *parser) forStmt() *forStmt {
	stmt := &forStmt{}
	for {
		p.consume(tokenIdentifier, "expect loop variable")
		stmt.vars = append(stmt.vars, p.previous.literal)
		if !p.match(tokenComma) {
			break
		}
	}
	p.consume(tokenIn, "expect 'in'")
	stmt.in = p.expr(precLowest)
	p.loopCtx = &loopCtx{p.loopCtx}
	stmt.loop = p.block()
	p.loopCtx = p.loopCtx.encl
	return stmt
}

func (p *parser) ifStmt() *ifStmt {
	stmt := &ifStmt{}
	stmt.cond = p.expr(precLowest)
	stmt.then = p.block()
	if p.match(tokenElif) {
		stmt.else_ = append(stmt.else_, p.ifStmt())
	} else if p.match(tokenElse) {
		stmt.else_ = p.block()
	} else {
		stmt.else_ = []astStmt{}
	}
	return stmt
}

func (p *parser) defStmt() *defStmt {
	stmt := &defStmt{}
	p.consume(tokenIdentifier, "expect function name")
	stmt.name = p.previous.literal
	p.consume(tokenLeftParen, "expect '('")
	stmt.params = p.params()
	p.defCtx = &defCtx{p.defCtx, nil}
	stmt.body = p.block()
	p.defCtx = p.defCtx.encl
	return stmt
}

func (p *parser) params() []string {
	params := []string{}
	if p.match(tokenRightParen) {
		return params
	}
	for {
		p.consume(tokenIdentifier, "expect parameter name")
		params = append(params, p.previous.literal)
		if !p.match(tokenComma) {
			break
		}
		if p.check(tokenRightParen) {
			break
		}
	}
	p.consume(tokenRightParen, "expect ')'")
	return params
}

func (p *parser) args() []astExpr {
	args := []astExpr{}
	if p.match(tokenRightParen) {
		return args
	}
	for {
		args = append(args, p.expr(precLowest))
		if !p.match(tokenComma) {
			break
		}
		if p.check(tokenRightParen) {
			break
		}
	}
	p.consume(tokenRightParen, "expect ')'")
	return args
}

func (p *parser) synchronize() {
	for p.current.tokenType != tokenEof {
		if p.previous.tokenType == tokenNewLine {
			return
		}
		switch p.current.tokenType {
		case tokenDef, tokenFor, tokenIf, tokenRaise, tokenTry,
			tokenWhile, tokenBreak, tokenContinue, tokenReturn, tokenExcept:
			return
		}

		p.advance()
	}
}
