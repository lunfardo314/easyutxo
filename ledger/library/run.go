package library

import (
	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

func NewRunContext(dataTree *lazyslice.Tree, path lazyslice.TreePath) *RunContext {
	return &RunContext{
		globalContext:  dataTree,
		invocationPath: path,
		CallStack:      make([]callFrame, maxCallDepth),
	}
}

func (glb *RunContext) Eval(f *easyfl.FormulaTree) []byte {
	glb.pushArgs(f.Args...)
	defer glb.popArgs()

	return f.EvalFunc(glb)
}

func (glb *RunContext) pushArgs(args ...*easyfl.FormulaTree) {
	glb.CallStack[glb.callStackTop] = args
	glb.callStackTop++
}

func (glb *RunContext) popArgs() {
	glb.callStackTop--
	glb.CallStack[glb.callStackTop] = nil
}

func (glb *RunContext) arg(n byte) []byte {
	return glb.Eval(glb.CallStack[glb.callStackTop-1][n])
}

func (glb *RunContext) arity() byte {
	return byte(len(glb.CallStack[glb.callStackTop-1]))
}
