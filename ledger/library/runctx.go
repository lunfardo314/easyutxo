package library

import (
	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

func NewRunContext(dataTree *lazyslice.Tree, path lazyslice.TreePath, data []byte) *RunContext {
	return &RunContext{
		globalContext:  dataTree,
		invocationPath: path,
		invocationData: data,
		evalStack:      make([]evalArgs, maxCallDepth),
		callStack:      make([]evalArgs, maxCallDepth),
	}
}

func (glb *RunContext) Eval(f *easyfl.FormulaTree) []byte {
	glb.pushEvalArgs(f.Args...)
	defer glb.popEvalArgs()

	return f.EvalFunc(glb)
}

// EvalWithArgs pushes values for argument references
func (glb *RunContext) EvalWithArgs(f *easyfl.FormulaTree, args ...[]byte) []byte {
	glb.pushCallArgs(easyfl.DataFormulas(args...))
	defer glb.popCallArgs()

	return glb.Eval(f)
}

func (glb *RunContext) pushEvalArgs(args ...*easyfl.FormulaTree) {
	glb.evalStack[glb.evalStackTop] = args
	glb.evalStackTop++
}

func (glb *RunContext) popEvalArgs() {
	glb.evalStackTop--
	glb.evalStack[glb.evalStackTop] = nil
}

func (glb *RunContext) pushCallArgs(args evalArgs) {
	glb.callStack[glb.callStackTop] = args
	glb.callStackTop++
}

func (glb *RunContext) popCallArgs() {
	glb.popCallBaseline()
}

func (glb *RunContext) pushCallBaseline() {
	glb.pushCallArgs(glb.evalStack[glb.evalStackTop-1])
}

func (glb *RunContext) popCallBaseline() {
	glb.callStackTop--
	glb.callStack[glb.callStackTop] = nil
}

func (glb *RunContext) arg(n byte) []byte {
	return glb.Eval(glb.evalStack[glb.evalStackTop-1][n])
}

func (glb *RunContext) arity() byte {
	return byte(len(glb.evalStack[glb.evalStackTop-1]))
}

func (glb *RunContext) callArg(n byte) []byte {
	return glb.Eval(glb.callStack[glb.callStackTop-1][n])
}
