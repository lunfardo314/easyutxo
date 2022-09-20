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

const maxCallDepth = 10

type RunContext struct {
	globalContext  *lazyslice.Tree
	invocationPath lazyslice.TreePath
	CallStack      []callFrame
	callStackTop   int
}

type callFrame [][]byte

const maxStack = 20

type (
	getRunnerFunc func(callArity byte) easyfl.EvalFunction
	runnerFunc    func(ctx *RunContext, args []*easyfl.FormulaTree) []byte
)
