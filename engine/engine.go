package engine

import (
	"github.com/lunfardo314/easyutxo/lazyslice"
)

const (
	NumRegisters = 256
	MaxStack     = 100
)

type (
	Engine interface {
		Push(data []byte)
		Pop()
	}
	engine struct {
		opcodes   Opcodes
		registers [NumRegisters][]byte
		stack     [MaxStack][]byte
		stackTop  int
		ctx       *lazyslice.Tree
	}
	OpCode interface {
		Bytes() []byte
		Uint16() uint16
		String() string
		Name() string
	}
	Opcodes interface {
		// ParseInstruction return first parsed instruction and remaining remainingCode
		ParseInstruction(code []byte) (InstructionRunner, []byte)
	}

	InstructionParser func(codeAfterOpcode []byte) (InstructionRunner, []byte)
	InstructionRunner func(e Engine) bool
)

const (
	RegInvocationPath = byte(iota)
	RegRemainingCode
)

// Run executes the script. If it returns, script is successful.
// If it panics, ledger is invalid
// invocationFullPath starts from validation root:
//  (a) (inputs, idx1, idx0, idxInsideOutput)
//  (b) (tx, context, idx, idxInsideOutput)
func Run(opcodes Opcodes, ctx *lazyslice.Tree, invocationFullPath lazyslice.TreePath, code, data []byte) {
	e := engine{
		opcodes: opcodes,
		ctx:     ctx,
	}
	e.registers[RegInvocationPath] = invocationFullPath
	e.registers[RegRemainingCode] = code
	e.Push(data)
	for e.run1Cycle() {
	}
}

func (e *engine) Push(data []byte) {
	e.stack[e.stackTop] = data
	e.stackTop++
}

func (e *engine) Pop() {
	if e.stackTop == 0 {
		panic("Pop: stack is empty")
	}
	e.stackTop--
	e.stack[e.stackTop] = nil // for GC
}

func (e *engine) run1Cycle() bool {
	var instrRunner InstructionRunner
	instrRunner, e.registers[RegRemainingCode] = e.opcodes.ParseInstruction(e.registers[RegRemainingCode])
	return instrRunner(e)
}
