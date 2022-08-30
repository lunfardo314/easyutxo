package engine

import "github.com/lunfardo314/easyutxo/lazyslice"

type engine struct {
	stack          [][]byte
	ctx            *lazyslice.Tree
	scriptLocation []byte
	remainingCode  []byte
}

// Run executes the script. If it returns, script is successful.
// If it panics, transaction is invalid
func Run(globalCtx *lazyslice.Tree, scriptPath ...byte) {
	e := engine{
		stack:          make([][]byte, 0),
		ctx:            globalCtx,
		scriptLocation: scriptPath,
		remainingCode:  globalCtx.BytesAtPath(scriptPath...),
	}
	for e.run1Cycle() {
	}
}

func (e *engine) run1Cycle() bool {
	var instrRunner instructionRunner
	instrRunner, e.remainingCode = parseInstruction(e.remainingCode)
	return instrRunner(e.ctx)
}
