package library

import (
	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

func NewRunContext(dataTree *lazyslice.Tree, path lazyslice.TreePath) *RunContext {
	return &RunContext{
		globalContext:  dataTree,
		invocationPath: path,
		EvalStack:      make([]evalArgs, maxCallDepth),
		CallStack:      make([]evalArgs, maxCallDepth),
	}
}

func (glb *RunContext) Eval(f *easyfl.FormulaTree) []byte {
	glb.pushEvalArgs(f.Args...)
	defer glb.popEvalArgs()

	return f.EvalFunc(glb)
}

func (glb *RunContext) EvalWithArgs(f *easyfl.FormulaTree, args ...[]byte) []byte {
	glb.pushCallArgs(easyfl.DataFormulas(args...))
	defer glb.popCallArgs()

	return glb.Eval(f)
}

func (glb *RunContext) pushEvalArgs(args ...*easyfl.FormulaTree) {
	glb.EvalStack[glb.evalStackTop] = args
	glb.evalStackTop++
}

func (glb *RunContext) popEvalArgs() {
	glb.evalStackTop--
	glb.EvalStack[glb.evalStackTop] = nil
}

func (glb *RunContext) pushCallArgs(args evalArgs) {
	glb.CallStack[glb.callStackTop] = args
	glb.callStackTop++
}

func (glb *RunContext) popCallArgs() {
	glb.popCallBaseline()
}

func (glb *RunContext) pushCallBaseline() {
	glb.pushCallArgs(glb.EvalStack[glb.evalStackTop-1])
}

func (glb *RunContext) popCallBaseline() {
	glb.callStackTop--
	glb.CallStack[glb.callStackTop] = nil
}

func (glb *RunContext) arg(n byte) []byte {
	return glb.Eval(glb.EvalStack[glb.evalStackTop-1][n])
}

func (glb *RunContext) arity() byte {
	return byte(len(glb.EvalStack[glb.evalStackTop-1]))
}

func (glb *RunContext) callArg(n byte) []byte {
	return glb.Eval(glb.CallStack[glb.callStackTop-1][n])
}
