package fylin

import (
	"fmt"
	"log"
	"math"
)

type env struct {
	store map[varName]Value
	types map[varName]varType
	encl  *env
}

func newEnv(encl *env) *env {
	return &env{
		store: make(map[varName]Value),
		encl:  encl,
		types: make(map[varName]varType),
	}
}

type Evaluator struct {
	Globals map[varName]Value
	*env
}

func New() *Evaluator {
	return &Evaluator{
		Globals: map[varName]Value{
			"println": &nativePrintln,
		},
		env: newEnv(nil),
	}
}

func (e *Evaluator) Interpret(source []byte) (err error) {
	defer catch(func(exc runtimeException) {
		fmt.Println(exc)
		err = exc
	})
	p := newParser(source)
	ast, err := p.parse()
	if err != nil {
		log.Fatal(err)
	}
	if debugPrintAST {
		p := &printer{}
		fmt.Println(cover("ast", 12, "="))
		fmt.Println(p.sprintProgram(ast))
		fmt.Println(cover("runtime", 12, "="))
	}
	for _, stmt := range ast {
		e.eval(stmt)
	}
	return
}

func (e *Evaluator) Call(function *Func) (val Value, err error) {
	defer catch(func(exc runtimeException) { err = exc })
	for _, stmt := range function.Code {
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
		e.set(
			&ident{node.name},
			&Func{
				Name:    node.name,
				Params:  node.params,
				Code:    node.body,
				Closure: e.env,
			},
		)
		return nil
	case *lambdaLit:
		return &Func{
			Name:    node.name,
			Params:  node.params,
			Code:    node.body,
			Closure: e.env,
		}
	case *decoStmt:
		deco, ok := e.eval(node.deco).(*Func)
		if !ok {
			Raise(Str("decorator must be function"))
		}
		def := &Func{
			Name:    node.def.name,
			Params:  node.def.params,
			Code:    node.def.body,
			Closure: e.env,
		}
		decored := e.call(deco, []Value{def})
		e.set(&ident{node.def.name}, decored)
		return nil
	case *dictLit:
		doc := &Doc{
			Pairs: make(map[Value]Value, len(node.pairs)),
			Proto: nil,
		}
		for key, val := range node.pairs {
			doc.Pairs[e.eval(key)] = e.eval(val)
		}
		return doc
	case *protoDictExpr:
		proto, ok := e.eval(node.proto).(*Doc)
		if !ok {
			Raise(Str("prototype must be 'doc' type"))
		}
		doc := &Doc{
			Pairs: make(map[Value]Value, len(node.dict.pairs)),
			Proto: proto,
		}
		for key, val := range node.dict.pairs {
			doc.Pairs[e.eval(key)] = e.eval(val)
		}
		return doc
	case *listLit:
		doc := &Doc{
			Pairs: make(map[Value]Value, len(node.elems)),
			Proto: &protoArray,
		}
		for key, val := range node.elems {
			doc.Pairs[Num(key)] = e.eval(val)
		}
		return doc
	case *infixExpr:
		l, r := e.eval(node.left), e.eval(node.right)
		switch node.opToken.tokenType {
		case tokenEqual:
			return valuesEqual(l, r)
		case tokenBangEqual:
			return !valuesEqual(l, r)
		case tokenLess, tokenLessEqual, tokenGreater, tokenGreaterEqual,
			tokenMinus, tokenStar, tokenSlash, tokenStarStar, tokenSlashSlash:
			return numberOperation(l, r, node.opToken.tokenType)
		case tokenPlus:
			return operation(l, r, node.opToken.tokenType)
		default:
			panic("eval infix expr: unknown operation")
		}
	case *whileStmt:
		e.whileStmt(node)
		return nil
	case *tryStmt:
		e.tryStmt(node)
		return nil
	case *exprStmt:
		e.eval(node.expr)
		return nil
	case *raiseStmt:
		Raise(e.eval(node.exc))
		return nil
	case *assignStmt:
		rightVals := []Value{}
		for _, r := range node.rights {
			rightVals = append(rightVals, e.eval(r))
		}
		if len(node.lefts) > len(rightVals) {
			for range len(node.lefts) - len(rightVals) {
				rightVals = append(rightVals, None{})
			}
		}
		for i, l := range node.lefts {
			e.set(l, rightVals[i])
		}
		return nil
	case *declStmt:
		for _, name := range node.vars {
			e.types[name] = node.varType
		}
		return nil
	case *ident:
		return e.get(node)
	case *callExpr:
		left := e.eval(node.left)
		args := e.evalExprs(node.args)
		return e.call(left, args)
	case *returnStmt:
		panic(returnSignal(e.evalExprs(node.values)))
	case *ifStmt:
		if valueToBool(e.eval(node.cond)) {
			for _, stmt := range node.then {
				e.eval(stmt)
			}
		} else {
			for _, stmt := range node.else_ {
				e.eval(stmt)
			}
		}
		return nil
	case *indexExpr:
		return e.get(node)
	}
	panic("eval: unknown node type")
}

func (e *Evaluator) call(callee Value, args []Value) Value {
	native, ok := callee.(*NativeFunc)
	if ok {
		return native.Code(e, args)
	}

	called, ok := callee.(*Func)
	if !ok {
		Raise(Str("can only call functions"))
	}

	if len(args) < len(called.Params) {
		for range len(called.Params) - len(args) {
			args = append(args, None{})
		}
	}

	encl := e.env
	e.env = newEnv(called.Closure)
	for i, param := range called.Params {
		e.env.store[param] = args[i]
	}

	ret := e.evalCode(called.Code)

	e.env = encl

	if ret != nil {
		return ret[0]
	}
	return None{}
}

func (e *Evaluator) whileStmt(node *whileStmt) {
	defer catch(func(sig breakSignal) {})

	loop := func() {
		defer catch(func(sig continueSignal) {})

		for _, stmt := range node.loop {
			e.eval(stmt)
		}
	}

	for valueToBool(e.eval(node.cond)) {
		loop()
	}
}

func (e *Evaluator) tryStmt(node *tryStmt) {
	var exc Value
	func() {
		defer catch(func(sig runtimeException) { exc = sig.value })

		for _, stmt := range node.try {
			e.eval(stmt)
		}
	}()

	var reRaise Value
	func() {
		defer catch(func(sig runtimeException) { reRaise = sig.value })

		if exc != nil {
			e.set(&ident{node.as}, exc)
			for _, stmt := range node.except {
				e.eval(stmt)
			}
		}
	}()

	for _, stmt := range node.finally {
		e.eval(stmt)
	}

	if reRaise != nil {
		Raise(reRaise)
	}
}

func (e *Evaluator) evalCode(stmts []astStmt) (vals []Value) {
	defer catch(func(sig returnSignal) { vals = sig })

	for _, stmt := range stmts {
		e.eval(stmt)
	}
	return
}

type returnSignal []Value
type continueSignal struct{}
type breakSignal struct{}

func (e *Evaluator) evalExprs(exprs []astExpr) []Value {
	ret := []Value{}
	for _, arg := range exprs {
		ret = append(ret, e.eval(arg))
	}
	return ret
}

func (e *Evaluator) set(to astExpr, val Value) {
	switch to := to.(type) {
	case *ident:
		if t, ok := e.env.types[to.name]; ok {
			switch t {
			case varLocal:
				e.env.store[to.name] = val
				return
			case varNonLocal:
				env := e.env.encl
				for env != nil {
					if _, ok := env.store[to.name]; ok {
						env.store[to.name] = val
						return
					}
					env = env.encl
				}
				Raise(Str("undefined variable"))
			case varGlobal:
				e.Globals[to.name] = val
				return
			}
		}
		e.env.store[to.name] = val
	case *indexExpr:
		toDoc, ok := e.eval(to.left).(*Doc)
		if ok {
			toDoc.Pairs[e.eval(to.index)] = val
			return
		}
		toBox, ok := e.eval(to.left).(*Box)
		if !ok {
			Raise(Str("TODO"))
		}
		toBox.Set(e.eval(to.index), val)
	default:
		panic("set: unknown type")
	}
}

func (e *Evaluator) get(from astExpr) Value {
	switch from := from.(type) {
	case *ident:
		if t, ok := e.env.types[from.name]; ok {
			switch t {
			case varLocal:
				if v, ok := e.env.store[from.name]; ok {
					return v
				}
			case varNonLocal:
				env := e.env.encl
				for env != nil {
					if v, ok := env.store[from.name]; ok {
						return v
					}
					env = env.encl
				}
			case varGlobal:
				if v, ok := e.Globals[from.name]; ok {
					return v
				}
			}
			Raise(Str("undefined variable"))
		}
		env := e.env
		for env != nil {
			if v, ok := env.store[from.name]; ok {
				return v
			}
			env = env.encl
		}
		if v, ok := e.Globals[from.name]; ok {
			return v
		}
		Raise(Str("undefined variable"))
	case *indexExpr:
		toDoc, ok := e.eval(from.left).(*Doc)
		if ok {
			if v, ok := toDoc.Pairs[e.eval(from.index)]; ok {
				return v
			}
			return None{}
		}
		toBox, ok := e.eval(from.left).(*Box)
		if !ok {
			Raise(Str("TODO"))
		}
		return toBox.Get(e.eval(from.index))
	}
	panic("get: unknown type")
}

func valueToBool(val Value) Bool {
	if _, ok := val.(None); ok {
		return false
	}
	if bv, ok := val.(Bool); ok {
		return bv
	}
	return true
}

func valuesEqual(a, b Value) Bool { return a == b }

func numberOperation(a, b Value, op tokenType) Value {
	an, ok1 := a.(Num)
	bn, ok2 := b.(Num)
	if !ok1 || !ok2 {
		Raise(Str("operands must be numbers"))
	}
	switch op {
	case tokenPlus:
		return an + bn
	case tokenMinus:
		return an - bn
	case tokenStar:
		return an * bn
	case tokenSlash:
		return an / bn
	case tokenStarStar:
		return Num(math.Pow(float64(an), float64(bn)))
	case tokenSlashSlash:
		return Num(math.Floor(float64(an / bn)))
	case tokenLess:
		return Bool(an < bn)
	case tokenLessEqual:
		return Bool(an <= bn)
	case tokenGreater:
		return Bool(an > bn)
	case tokenGreaterEqual:
		return Bool(an >= bn)
	}
	panic("number operation: unknown operation")
}

func operation(a, b Value, op tokenType) Value {
	if op == tokenPlus {
		as, ok1 := a.(Str)
		bs, ok2 := b.(Str)
		if ok1 && ok2 {
			return as + bs
		}
		an, ok1 := a.(Num)
		bn, ok2 := b.(Num)
		if ok1 && ok2 {
			return an + bn
		}
		Raise(Str("operands must be numbers or strings"))
	}
	panic("operation: unknown operation")
}

type runtimeException struct {
	value Value
}

func (exc runtimeException) Error() string {
	return fmt.Sprint(exc.value)
}

func Raise(exception Value) {
	panic(runtimeException{exception})
}
