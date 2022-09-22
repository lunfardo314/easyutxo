package library

import (
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
	globalContext  *lazyslice.Tree
	invocationPath lazyslice.TreePath
	invocationData []byte
	evalStack      []evalArgs
	evalStackTop   int
	callStack      []evalArgs
	callStackTop   int
}

type evalArgs []*easyfl.FormulaTree

type runnerFunc func(ctx *RunContext) []byte
