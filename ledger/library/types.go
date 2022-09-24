package library

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

type funDescriptor struct {
	sym               string
	funCode           uint16
	requiredNumParams int
	evalFun           easyfl.EvalFunction
}

const maxCallDepth = 30

type GlobalContext struct {
	dataTree       *lazyslice.Tree
	invocationPath lazyslice.TreePath
}

type evalArgs []*easyfl.FormulaTree

type (
	EvalFunc             func(ctx *easyfl.RunContext) []byte
	InvokeConstraintFunc func(*lazyslice.Tree, lazyslice.TreePath) []byte
)

var invokeConstraint InvokeConstraintFunc

func RegisterInvokeConstraintFunc(f InvokeConstraintFunc) {
	invokeConstraint = f
}

func MustMakeEvalFunc(formulaSource string) EvalFunc {
	f, _, _, err := easyfl.CompileFormula(Library, formulaSource)
	if err != nil {
		panic(fmt.Errorf("MustMakeEvalFunc: '%v' -- '%s'", err, formulaSource))
	}
	return func(ctx *easyfl.RunContext) []byte {
		return ctx.Eval(f)
	}
}
