package fylin

import (
	"fmt"
	"strconv"
)

type Value interface {
	fyValue()
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

type NativeFunc struct {
	Code func(e *Evaluator, args []Value) Value
	Name string
}

type Doc struct {
	Pairs map[Value]Value
	Proto *Doc
}

type Box struct {
	Set func(index Value, value Value)
	Get func(index Value) Value
}

func (v None) fyValue()        {}
func (v Bool) fyValue()        {}
func (v Num) fyValue()         {}
func (v Str) fyValue()         {}
func (v *Doc) fyValue()        {}
func (v *Func) fyValue()       {}
func (v *NativeFunc) fyValue() {}
func (v *Box) fyValue()        {}

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

var nativePrintln = NativeFunc{
	Name: "println",
	Code: func(e *Evaluator, args []Value) Value {
		for i, arg := range args {
			fmt.Print(arg)
			if i != len(args)-1 {
				fmt.Print(" ")
			}
		}
		fmt.Println()
		return None{}
	},
}

var protoArray = Doc{map[Value]Value{}, nil}
