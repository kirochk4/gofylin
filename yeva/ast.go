package yeva

import (
	"fmt"
	"strconv"
	"strings"
)

type astNode interface {
	astNode()
}

type astStmt interface {
	astNode
	astStmt()
}

type astExpr interface {
	astNode
	astExpr()
}

/* == statements ============================================================ */

type badStmt string

type declStmt struct {
	varType
	vars []varName
}

type decoStmt struct {
	deco astExpr
	def  *defStmt
}

type defStmt struct {
	name   string
	params []varName
	body   []astStmt
}

type exprStmt struct {
	expr astExpr
}

type raiseStmt struct {
	exc astExpr
}

type tryStmt struct {
	try     []astStmt
	except  []astStmt // can be nil
	as      varName
	finally []astStmt // can be nil
}

type ifStmt struct {
	cond  astExpr
	then  []astStmt
	else_ []astStmt
}

type forStmt struct {
	loop []astStmt
	vars []varName
	in   astExpr
}

type whileStmt struct {
	loop []astStmt
	cond astExpr
}

type returnStmt struct {
	values []astExpr
}

type breakStmt struct{}

type continueStmt struct{}

type assignStmt struct {
	lefts  []astExpr
	rights []astExpr
}

/* == expression ============================================================ */

type infixExpr struct {
	left    astExpr
	right   astExpr
	opToken token
}

type prefixExpr struct {
	right   astExpr
	opToken token
}

type callExpr struct {
	left astExpr
	args []astExpr
}

type indexExpr struct {
	left  astExpr
	index astExpr
}

type arrowExpr struct {
	left  astExpr
	index astExpr
}

type protoDictExpr struct {
	proto astExpr
	dict  *dictLit
}

type ident struct {
	name varName
}

type noneLit struct{}

type boolLit struct {
	value bool
}

type numLit struct {
	value float64
}

type strLit struct {
	value string
}

type dictLit struct {
	pairs map[astExpr]astExpr
}

type listLit struct {
	elems []astExpr
}

type lambdaLit struct {
	*defStmt
}

/* == marks ================================================================= */

func (n badStmt) astStmt()       {}
func (n *decoStmt) astStmt()     {}
func (n *defStmt) astStmt()      {}
func (n *exprStmt) astStmt()     {}
func (n *ifStmt) astStmt()       {}
func (n *forStmt) astStmt()      {}
func (n *whileStmt) astStmt()    {}
func (n *returnStmt) astStmt()   {}
func (n *breakStmt) astStmt()    {}
func (n *continueStmt) astStmt() {}
func (n *assignStmt) astStmt()   {}
func (n *raiseStmt) astStmt()    {}
func (n *tryStmt) astStmt()      {}
func (n *declStmt) astStmt()     {}

func (n *infixExpr) astExpr()     {}
func (n *prefixExpr) astExpr()    {}
func (n *callExpr) astExpr()      {}
func (n *indexExpr) astExpr()     {}
func (n *arrowExpr) astExpr()     {}
func (n *protoDictExpr) astExpr() {}
func (n *ident) astExpr()         {}
func (n *noneLit) astExpr()       {}
func (n *boolLit) astExpr()       {}
func (n *numLit) astExpr()        {}
func (n *strLit) astExpr()        {}
func (n *dictLit) astExpr()       {}
func (n *listLit) astExpr()       {}
func (n *lambdaLit) astExpr()     {}

func (n badStmt) astNode()       {}
func (n *decoStmt) astNode()     {}
func (n *defStmt) astNode()      {}
func (n *exprStmt) astNode()     {}
func (n *ifStmt) astNode()       {}
func (n *forStmt) astNode()      {}
func (n *whileStmt) astNode()    {}
func (n *returnStmt) astNode()   {}
func (n *breakStmt) astNode()    {}
func (n *continueStmt) astNode() {}
func (n *assignStmt) astNode()   {}
func (n *raiseStmt) astNode()    {}
func (n *tryStmt) astNode()      {}
func (n *declStmt) astNode()     {}

func (n *infixExpr) astNode()     {}
func (n *prefixExpr) astNode()    {}
func (n *callExpr) astNode()      {}
func (n *indexExpr) astNode()     {}
func (n *arrowExpr) astNode()     {}
func (n *protoDictExpr) astNode() {}
func (n *ident) astNode()         {}
func (n *noneLit) astNode()       {}
func (n *boolLit) astNode()       {}
func (n *numLit) astNode()        {}
func (n *strLit) astNode()        {}
func (n *dictLit) astNode()       {}
func (n *listLit) astNode()       {}
func (n *lambdaLit) astNode()     {}

/* == print ================================================================= */

const tabPrintSize = 4

type printer struct {
	tab  int
	data *strings.Builder
}

func (p *printer) sprintProgram(program []astStmt) string {
	p.tab = 0
	p.data = &strings.Builder{}
	for i, stmt := range program {
		p.writeNode(stmt)
		if i != len(program)-1 {
			p.write("\n")
		}
	}
	return p.data.String()
}

func (p *printer) writeNode(node astNode) {
	switch node := node.(type) {
	case *assignStmt:
		p.writeExprs(node.lefts)
		p.write(" = ")
		p.writeExprs(node.rights)
	case *returnStmt:
		p.write("return ")
		p.writeExprs(node.values)
	case *defStmt:
		p.write("def %s", node.name)
		p.writeParams(node.params)
		p.writeBlock(node.body)
	case *exprStmt:
		p.writeNode(node.expr)
	case *ifStmt:
		p.write("if ")
		p.writeNode(node.cond)
		p.writeBlock(node.then)
		p.write("\n")
		p.writeTab()
		p.write("else")
		p.writeBlock(node.else_)
	case *tryStmt:
		p.write("try")
		p.writeBlock(node.try)
		if node.except != nil {
			p.write("\n")
			p.writeTab()
			p.write("expect as ")
			p.write("%s", node.as)
			p.writeBlock(node.except)
		}
		if node.finally != nil {
			p.write("\n")
			p.writeTab()
			p.write("finally")
			p.writeBlock(node.finally)
		}
	case *decoStmt:
		p.write("@")
		p.writeNode(node.deco)
		p.write("\n")
		p.writeTab()
		p.writeNode(node.def)
	case *raiseStmt:
		p.write("raise ")
		p.writeNode(node.exc)
	case *whileStmt:
		p.write("while ")
		p.writeNode(node.cond)
		p.writeBlock(node.loop)
	case *infixExpr:
		p.writeNode(node.left)
		p.write(" %s ", node.opToken.literal)
		p.writeNode(node.right)
	case *prefixExpr:
		p.write("%s ", node.opToken.literal)
		p.writeNode(node.right)
	case *protoDictExpr:
		p.writeNode(node.proto)
		p.write("{ TODO }")
	case *indexExpr:
		p.writeNode(node.left)
		p.write("[")
		p.writeNode(node.index)
		p.write("]")
	case *callExpr:
		p.writeNode(node.left)
		p.writeArgs(node.args)
	case *declStmt:
		p.write("%s ", string(node.varType))
		p.writeVars(node.vars)
	case *arrowExpr:
		p.writeNode(node.left)
		p.write("->[")
		p.writeNode(node.index)
		p.write("]")

	case *noneLit:
		p.write("None")
	case *boolLit:
		if node.value {
			p.write("True")
		} else {
			p.write("False")
		}
	case *numLit:
		p.write("%s", strconv.FormatFloat(node.value, 'g', -1, 64))
	case *strLit:
		p.write("\"%s\"", node.value)
	case *dictLit:
		p.write("{ TODO }")
	case *listLit:
		p.write("[")
		p.writeExprs(node.elems)
		p.write("]")
	case *lambdaLit:
		p.write("lambda ")
		p.writeVars(node.params)
		p.write(": ")
		p.writeNode(node.body[0].(*returnStmt).values[0])

	case *ident:
		p.write("%s", node.name)
	default:
		p.write("(undefined)")
	}
}

func (p *printer) write(str string, a ...any) { fmt.Fprintf(p.data, str, a...) }
func (p *printer) writeTab()                  { p.write("%*s", p.tab, "") }
func (p *printer) addTab()                    { p.tab += tabPrintSize }
func (p *printer) subTab()                    { p.tab -= tabPrintSize }

func (p *printer) writeBlock(block []astStmt) {
	p.write(":\n")
	p.addTab()
	if len(block) == 0 {
		p.writeTab()
		p.write("pass")
	} else {
		for i, node := range block {
			p.writeTab()
			p.writeNode(node)
			if i != len(block)-1 {
				p.write("\n")
			}
		}
	}
	p.subTab()
}

func (p *printer) writeExprs(exprs []astExpr) {
	for i, expr := range exprs {
		p.writeNode(expr)
		if i != len(exprs)-1 {
			p.write(", ")
		}
	}
}

func (p *printer) writeVars(vars []varName) {
	for i, var_ := range vars {
		p.write("%s", var_)
		if i != len(vars)-1 {
			p.write(", ")
		}
	}
}

func (p *printer) writeArgs(args []astExpr) {
	p.write("(")
	for i, arg := range args {
		p.writeNode(arg)
		if i != len(args)-1 {
			p.write(", ")
		}
	}
	p.write(")")
}

func (p *printer) writeParams(params []varName) {
	p.write("(")
	for i, param := range params {
		p.write("%s", param)
		if i != len(params)-1 {
			p.write(", ")
		}
	}
	p.write(")")
}
