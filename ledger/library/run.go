package library

import (
	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

func NewRunContext(glb *lazyslice.Tree, path lazyslice.TreePath) *RunContext {
	return &RunContext{
		globalContext:  glb,
		invocationPath: path,
		CallStack:      make([]callFrame, maxCallDepth),
	}
}

func (glb *RunContext) Call(f *easyfl.FormulaTree, args ...[]byte) []byte {
	glb.pushArgs(args...)
	defer glb.popArgs()

	return f.Eval(glb)
}

func (glb *RunContext) pushArgs(args ...[]byte) {
	glb.CallStack[glb.callStackTop] = args
	glb.callStackTop++
}

func (glb *RunContext) popArgs() {
	glb.callStackTop--
	glb.CallStack[glb.callStackTop] = nil
}

func (glb *RunContext) arg(n byte) []byte {
	return glb.CallStack[glb.callStackTop-1][n]
}
