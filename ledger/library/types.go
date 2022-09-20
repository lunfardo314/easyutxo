package library

import (
	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

type funDescriptor struct {
	sym       string
	funCode   uint16
	numParams int
	evalFun   easyfl.EvalFunction
}

type RunContext struct {
	globalContext  *lazyslice.Tree
	invocationPath lazyslice.TreePath
}

const maxStack = 20

type (
	getRunnerFunc func(callArity byte) easyfl.EvalFunction
	runnerFunc    func(ctx *RunContext, args []*easyfl.FormulaTree) []byte
)
