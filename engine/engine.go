package engine

import (
	"github.com/lunfardo314/easyutxo/lazyslice"
)

const (
	NumRegisters = 256
	MaxStack     = 100
)

type engine struct {
	registers [NumRegisters][]byte
	stack     [MaxStack][]byte
	stackTop  int
	ctx       *lazyslice.Tree
}

const (
	RegInvocationPath = byte(iota)
	RegInvocationData
	RegRemainingCode
)

// Run executes the script. If it returns, script is successful.
// If it panics, transaction is invalid
// invocationFullPath starts from validation root:
//  (a) (inputs, idx1, idx0, idxInsideOutput)
//  (b) (tx, context, idx, idxInsideOutput)
func Run(ctx *lazyslice.Tree, invocationFullPath lazyslice.TreePath, code, data []byte) {
	e := engine{
		ctx: ctx,
	}
	e.registers[RegInvocationPath] = invocationFullPath
	e.registers[RegRemainingCode] = code
	e.registers[RegInvocationData] = data
	for e.run1Cycle() {
	}
}

func (e *engine) run1Cycle() bool {
	var instrRunner instructionRunner
	instrRunner, e.registers[RegRemainingCode] = parseInstruction(e.registers[RegRemainingCode])
	return instrRunner(e.ctx)
}
