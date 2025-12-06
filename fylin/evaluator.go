package fylin

import (
	"fmt"
	"math"
)

type returnSignal []Value
type continueSignal struct{}
type breakSignal struct{}

type runtimeException struct {
	value Value
}

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
		fmt.Println(exc) // remove this!
		err = exc
	})
	p := newParser(source)
	ast, err := p.parse()
	if err != nil {
		return fmt.Errorf("compile error: %w", err)
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

func (e *Evaluator) Call(callee Callable, args []Value) (vals []Value, err error) {
	defer catch(func(exc runtimeException) { err = exc })

	return callee.call(e, args), nil
}

func (e *Evaluator) evalOne(node astNode) Value {
	return e.eval(node)[0]
}

func (e *Evaluator) eval(node astNode) []Value {
	switch node := node.(type) {
	case *noneLit:
		return one(None{})
	case *boolLit:
		return one(Bool(node.value))
	case *numLit:
		return one(Num(node.value))
	case *strLit:
		return one(Str(node.value))
	case *defStmt:
		e.env.store[node.name] = &Func{
			Name:    node.name,
			Params:  node.params,
			Code:    node.body,
			Closure: e.env,
		}
		return nil
	case *lambdaLit:
		return one(&Func{
			Name:    node.name,
			Params:  node.params,
			Code:    node.body,
			Closure: e.env,
		})
	case *decoStmt:
		deco, ok := e.evalOne(node.deco).(*Func)
		if !ok {
			Raise(Str("decorator must be function"))
		}
		def := &Func{
			Name:    node.def.name,
			Params:  node.def.params,
			Code:    node.def.body,
			Closure: e.env,
		}
		decored := deco.call(e, one(def))[0]
		e.env.store[node.def.name] = decored
		return nil
	case *dictLit:
		doc := &Doc{
			Pairs: make(map[Value]Value, len(node.pairs)),
			Proto: nil,
		}
		for key, val := range node.pairs {
			v := e.evalOne(val)
			k := e.evalOne(key)
			if _, none := k.(None); none {
				continue
			}
			doc.Pairs[k] = v
		}
		return one(doc)
	case *protoDictExpr:
		proto, ok := e.evalOne(node.proto).(Prototype)
		if !ok {
			Raise(Str("wrong prototype type"))
		}
		doc := &Doc{
			Pairs: make(map[Value]Value, len(node.dict.pairs)),
			Proto: &proto,
		}
		for key, val := range node.dict.pairs {
			v := e.evalOne(val)
			k := e.evalOne(key)
			if _, none := k.(None); none {
				continue
			}
			doc.Pairs[k] = v
		}
		return one(doc)
	case *listLit:
		doc := &Doc{
			Pairs: make(map[Value]Value, len(node.elems)),
			Proto: &protoArray,
		}
		for key, val := range node.elems {
			doc.Pairs[Num(key)] = e.evalOne(val) // maybe many?
		}
		return one(doc)
	case *infixExpr:
		l, r := e.evalOne(node.left), e.evalOne(node.right)
		switch node.opToken.tokenType {
		case tokenEqual:
			return one(valuesEqual(l, r))
		case tokenBangEqual:
			return one(!valuesEqual(l, r))
		case tokenLess, tokenLessEqual, tokenGreater, tokenGreaterEqual,
			tokenMinus, tokenStar, tokenSlash, tokenStarStar, tokenSlashSlash:
			return one(numberOperation(l, r, node.opToken.tokenType))
		case tokenPlus:
			return one(operation(l, r, node.opToken.tokenType))
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
		Raise(e.evalOne(node.exc))
		return nil
	case *assignStmt:
		rightVals := []Value{}
		for i, r := range node.rights {
			if i != len(node.rights)-1 {
				rightVals = append(rightVals, e.evalOne(r))
			} else {
				rightVals = append(rightVals, e.eval(r)...)
			}
		}
		if len(node.lefts) > len(rightVals) {
			for range len(node.lefts) - len(rightVals) {
				rightVals = append(rightVals, None{})
			}
		}
		for i, l := range node.lefts {
			e.assign(l, rightVals[i])
		}
		return nil
	case *declStmt:
		for _, name := range node.vars {
			e.types[name] = node.varType
		}
		return nil
	case *ident:
		return one(e.resolveVariable(node))
	case *callExpr:
		left, ok := e.evalOne(node.left).(Callable)
		if !ok {
			Raise(Str("call not collable"))
		}
		args := e.evalExprs(node.args)
		return left.call(e, args)
	case *returnStmt:
		panic(returnSignal(e.evalExprs(node.values)))
	case *ifStmt:
		if valueToBool(e.evalOne(node.cond)) {
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
		index := e.evalOne(node.index)
		from, ok := e.evalOne(node.left).(Prototype)
		if !ok {
			Raise(Str("can't get index"))
		}
		return one(from.Index(index))
	case *arrowExpr:
		index := e.evalOne(node.index)
		from, ok := e.evalOne(node.left).(Prototype)
		if !ok {
			Raise(Str("can't get index"))
		}
		if from.Prototype() != nil {
			v := (*from.Prototype()).Index(index)
			if f, ok := v.(Callable); ok {
				v = &Method{from.(Value), f}
			}
			return one(v)
		}
		return one(None{})
	}
	panic("eval: unknown node type")
}

func (e *Evaluator) whileStmt(node *whileStmt) {
	defer catch(func(sig breakSignal) {})

	loop := func() {
		defer catch(func(sig continueSignal) {})

		for _, stmt := range node.loop {
			e.eval(stmt)
		}
	}

	for valueToBool(e.evalOne(node.cond)) {
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
			e.env.store[node.as] = exc
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

func (e *Evaluator) evalExprs(exprs []astExpr) []Value {
	ret := []Value{}
	for i, arg := range exprs {
		if i != len(exprs)-1 {
			ret = append(ret, e.evalOne(arg))
		} else {
			ret = append(ret, e.eval(arg)...)
		}
	}
	return ret
}

func (e *Evaluator) resolveVariable(ident *ident) Value {
	if t, ok := e.env.types[ident.name]; ok {
		switch t {
		case varLocal:
			if v, ok := e.env.store[ident.name]; ok {
				return v
			}
		case varNonLocal:
			env := e.env.encl
			for env != nil {
				if v, ok := env.store[ident.name]; ok {
					return v
				}
				env = env.encl
			}
		case varGlobal:
			if v, ok := e.Globals[ident.name]; ok {
				return v
			}
		}
		Raise(Str("undefined variable"))
	}

	env := e.env
	for env != nil {
		if v, ok := env.store[ident.name]; ok {
			return v
		}
		env = env.encl
	}
	if v, ok := e.Globals[ident.name]; ok {
		return v
	}
	Raise(Str("undefined variable"))
	return nil
}

func (e *Evaluator) assign(to astExpr, val Value) {
	switch to := to.(type) {
	case *ident:
		e.assignToVariable(to.name, val)
	case *indexExpr:
		index := e.evalOne(to.index)
		left := e.evalOne(to.left)
		switch left := left.(type) {
		case *Doc:
			if _, del := val.(None); del {
				delete(left.Pairs, index)
				return
			}
			left.Pairs[index] = val
		case *Box:
			left.Setter(index, val)
		default:
			Raise(Str("TODO"))
		}
	default:
		panic("set: unknown type")
	}
}

func (e *Evaluator) assignToVariable(variable varName, val Value) {
	if t, ok := e.env.types[variable]; ok {
		switch t {
		case varLocal:
			e.env.store[variable] = val
			return
		case varNonLocal:
			env := e.env.encl
			for env != nil {
				if _, ok := env.store[variable]; ok {
					env.store[variable] = val
					return
				}
				env = env.encl
			}
		case varGlobal:
			e.Globals[variable] = val
			return
		}
		Raise(Str("undefined variable"))
	}

	e.env.store[variable] = val
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

func (exc runtimeException) Error() string {
	return fmt.Sprint(exc.value)
}

func Raise(exception Value) {
	panic(runtimeException{exception})
}

func one(val Value) []Value { return []Value{val} }
