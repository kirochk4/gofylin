package fylin

import (
	"fmt"
	"strconv"
)

type Value interface {
	fyValue()
}

type Prototype interface {
	Value
	Index(key Value) Value
	Prototype() *Prototype
}

type Callable interface {
	Value
	call(e *Evaluator, args []Value) []Value
}

type None struct{}

type Bool bool

type Num float64

type Str string

type Func struct {
	Code    []astStmt
	Params  []varName
	Closure *env
	Name    string
}

func (f *Func) call(e *Evaluator, args []Value) []Value {
	if len(args) < len(f.Params) {
		for range len(f.Params) - len(args) {
			args = append(args, None{})
		}
	}

	encl := e.env
	e.env = newEnv(f.Closure)
	for i, param := range f.Params {
		e.env.store[param] = args[i]
	}

	var ret []Value
	func() {
		defer catch(func(sig returnSignal) { ret = sig })

		for _, stmt := range f.Code {
			e.eval(stmt)
		}
	}()

	e.env = encl

	if ret != nil {
		return ret
	}
	return one(None{})
}

type NativeFunc struct {
	Code func(e *Evaluator, args []Value) []Value
	Name string
}

func (nf *NativeFunc) call(e *Evaluator, args []Value) []Value {
	return nf.Code(e, args)
}

type Method struct {
	self   Value
	method Callable
}

func (m *Method) call(e *Evaluator, args []Value) []Value {
	args = append([]Value{m.self}, args...)
	return m.method.call(e, args)
}

type Doc struct {
	Pairs map[Value]Value
	Proto *Prototype
}

func (d *Doc) Index(key Value) Value {
	v, ok := d.Pairs[key]
	if ok {
		return v
	}
	if d.Proto != nil {
		return (*d.Proto).Index(key)
	}
	return None{}
}

func (d *Doc) Prototype() *Prototype {
	return d.Proto
}

type Box struct {
	Setter func(key Value, value Value)
	Getter func(key Value) Value
	Proto  *Prototype
}

func (b *Box) Index(key Value) Value {
	v := b.Getter(key)
	if !isNone(v) {
		return v
	}
	if b.Proto != nil {
		return (*b.Proto).Index(key)
	}
	return None{}
}

func (b *Box) Prototype() *Prototype {
	return b.Proto
}

func (v None) fyValue()        {}
func (v Bool) fyValue()        {}
func (v Num) fyValue()         {}
func (v Str) fyValue()         {}
func (v *Doc) fyValue()        {}
func (v *Func) fyValue()       {}
func (v *NativeFunc) fyValue() {}
func (v *Box) fyValue()        {}
func (v *Method) fyValue()     {}

func (v None) String() string { return "None" }
func (v Bool) String() string {
	if v {
		return "True"
	}
	return "False"
}
func (v Num) String() string {
	return strconv.FormatFloat(float64(v), 'g', -1, 64)
}
func (v Str) String() string         { return string(v) }
func (v *Doc) String() string        { return "[doc Doc]" }
func (v *Func) String() string       { return "[func Func]" }
func (v *NativeFunc) String() string { return "[native Func]" }
func (v *Box) String() string        { return "[box Box]" }
func (v *Method) String() string     { return fmt.Sprint(v.method) }

var nativePrintln = NativeFunc{
	Name: "println",
	Code: func(e *Evaluator, args []Value) []Value {
		for i, arg := range args {
			fmt.Print(arg)
			if i != len(args)-1 {
				fmt.Print(" ")
			}
		}
		fmt.Println()
		return one(None{})
	},
}

var protoArray Prototype = &Doc{map[Value]Value{}, nil}

func isNone(val Value) bool {
	_, ok := val.(None)
	return ok
}
