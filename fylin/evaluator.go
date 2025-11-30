package fylin

import (
	"fmt"
	"log"
)

type Evaluator struct {
	Globals map[varName]Value
	module  map[varName]Value
}

func New() *Evaluator {
	return &Evaluator{
		Globals: make(map[varName]Value),
		module:  make(map[varName]Value),
	}
}

func (e *Evaluator) Interpret(source []byte) (err error) {
	defer catch(func(exc runtimeException) { err = exc })
	p := newParser(source)
	ast, err := p.parse()
	if err != nil {
		log.Fatal(err)
	}
	if debugPrintAST {
		p := &printer{}
		fmt.Println(cover("ast", 12, "="))
		fmt.Println(p.sprintProgram(ast))
	}
	for _, stmt := range ast {
		e.eval(stmt)
	}
	return
}

func (e *Evaluator) Call(function *Func) (val Value, err error) {
	defer catch(func(exc runtimeException) { err = exc })
	for _, stmt := range function.code {
		e.eval(stmt)
	}
	return None{}, nil
}

func (e *Evaluator) eval(node astNode) Value {
	switch node := node.(type) {
	case *noneLit:
		return None{}
	case *boolLit:
		return Bool(node.value)
	case *numLit:
		return Num(node.value)
	case *strLit:
		return Str(node.value)
	case *defStmt:
		return &Func{
			name:    node.name,
			params:  node.params,
			code:    node.body,
			closure: e.module,
		}
	case *dictLit:
		obj := &Doc{
			pairs: make(map[Value]Value, len(node.pairs)),
			proto: nil,
		}
		for key, val := range node.pairs {
			obj.pairs[e.eval(key)] = e.eval(val)
		}
		return obj
	case *protoDictExpr:
		proto, ok := e.eval(node.proto).(*Doc)
		if !ok {
			Raise(Str("prototype must be 'obj' type"))
		}
		obj := &Doc{
			pairs: make(map[Value]Value, len(node.dict.pairs)),
			proto: proto,
		}
		for key, val := range node.dict.pairs {
			obj.pairs[e.eval(key)] = e.eval(val)
		}
		return obj
	case *infixExpr:
		l, r := e.eval(node.left), e.eval(node.right)
		switch node.opToken.tokenType {
		case tokenEqual:
			return valuesEqual(l, r)
		case tokenBangEqual:
			return !valuesEqual(l, r)
		case tokenLess, tokenLessEqual, tokenGreater, tokenGreaterEqual,
			tokenMinus, tokenStar, tokenSlash:
			return numberOperation(l, r, node.opToken.tokenType)
		case tokenPlus:
			return operation(l, r, node.opToken.tokenType)
		default:
			panic("eval infix expr: unknown operation")
		}
	default:
		panic("eval: unknown node type")
	}
}

func valuesEqual(a, b Value) Bool
func numberOperation(a, b Value, op tokenType) Value
func operation(a, b Value, op tokenType) Value

type runtimeException struct {
	value Value
}

func (exc runtimeException) Error() string {
	return "TODO"
}

func Raise(exception Value) {
	panic(runtimeException{exception})
}
