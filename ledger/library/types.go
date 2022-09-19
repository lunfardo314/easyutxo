package library

import (
	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

type funDescriptor struct {
	sym       string
	funCode   uint16
	numParams int
	getRunner getRunnerFunc
}

type RunContext struct {
	globalContext  *lazyslice.Tree
	invocationPath lazyslice.TreePath
	stack          [][]byte
	stackTop       int
	codeRunner     easyfl.CodeReader
}

const maxStack = 20

type (
	getRunnerFunc func(callArity byte) runnerFunc
	runnerFunc    func(*RunContext) []byte
)

func (ctx *RunContext) Pop() []byte {
	ret := ctx.stack[ctx.stackTop]
	ctx.stack[ctx.stackTop] = nil
	ctx.stackTop--
	return ret
}
