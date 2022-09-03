package engine

import "github.com/lunfardo314/easyutxo/lazyslice"

const (
	NumRegisters = 256
	MaxStack     = 100
)

type engine struct {
	registers      [NumRegisters][]byte
	stack          [MaxStack][]byte
	stackTop       int
	ctx            *lazyslice.Tree
	scriptLocation []byte
	remainingCode  []byte
}

// Run executes the script. If it returns, script is successful.
// If it panics, transaction is invalid
func Run(globalCtx *lazyslice.Tree, scriptPath lazyslice.TreePath) {
	e := engine{
		ctx:            globalCtx,
		scriptLocation: scriptPath,
		remainingCode:  globalCtx.BytesAtPath(scriptPath),
	}
	e.registers[0] = scriptPath
	for e.run1Cycle() {
	}
}

func (e *engine) run1Cycle() bool {
	var instrRunner instructionRunner
	instrRunner, e.remainingCode = parseInstruction(e.remainingCode)
	return instrRunner(e.ctx)
}
