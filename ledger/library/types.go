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

type RunContext struct {
	dataTree       *lazyslice.Tree
	invocationPath lazyslice.TreePath
	evalStack      []evalArgs
	evalStackTop   int
	callStack      []evalArgs
	callStackTop   int
}

type evalArgs []*easyfl.FormulaTree

type EvalFunc func(ctx *RunContext) []byte

func MustMakeEvalFunc(formulaSource string) EvalFunc {
	f, _, _, err := easyfl.CompileFormula(Library, formulaSource)
	if err != nil {
		panic(fmt.Errorf("MustMakeEvalFunc: '%v' -- '%s'", err, formulaSource))
	}
	return func(ctx *RunContext) []byte {
		return ctx.Eval(f)
	}
}
