package fylin

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
	closure map[varName]Value
	name    string
}

type Doc struct {
	pairs map[Value]Value
	proto *Doc
}

func (v None) fyValue()  {}
func (v Bool) fyValue()  {}
func (v Num) fyValue()   {}
func (v Str) fyValue()   {}
func (v *Func) fyValue() {}
func (v *Doc) fyValue()  {}
