package fylin

import "fmt"

type Value interface {
	fyValue()
}

type None struct{}

type Bool bool

type Num float64

type Str string

type Func struct {
	code    []astStmt
	params  []varName
	closure *env
	Name    string
}

type NativeFunc struct {
	code func(e *Evaluator, args []Value) Value
	Name string
}

type Doc struct {
	pairs map[Value]Value
	proto *Doc
}

func (v None) fyValue()        {}
func (v Bool) fyValue()        {}
func (v Num) fyValue()         {}
func (v Str) fyValue()         {}
func (v *Doc) fyValue()        {}
func (v *Func) fyValue()       {}
func (v *NativeFunc) fyValue() {}

var nativePrintln = NativeFunc{
	Name: "println",
	code: func(e *Evaluator, args []Value) Value {
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
