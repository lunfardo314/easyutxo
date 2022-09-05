package engine

import (
	"github.com/lunfardo314/easyutxo/lazyslice"
)

const (
	NumRegisters = 256
	MaxStack     = 100
)

type (
	Engine struct {
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
		ValidateOpcode(oc OpCode) error
	}
	InstructionParser func(codeAfterOpcode []byte) (InstructionRunner, []byte)
	InstructionRunner func(e *Engine, paramBytes []byte) bool
)

const (
	RegInvocationPath = byte(iota)
	RegInvocationData
	RegRemainingCode
)

// Run executes the script. If it returns, script is successful.
// If it panics, ledger is invalid
// invocationFullPath starts from validation root:
//  (a) (inputs, idx1, idx0, idxInsideOutput)
//  (b) (tx, context, idx, idxInsideOutput)
func Run(opcodes Opcodes, ctx *lazyslice.Tree, invocationFullPath lazyslice.TreePath, code, data []byte) {
	e := Engine{
		opcodes: opcodes,
		ctx:     ctx,
	}
	e.registers[RegInvocationPath] = invocationFullPath
	e.registers[RegInvocationData] = data
	e.registers[RegRemainingCode] = code
	for e.run1Cycle() {
	}
}

func (e *Engine) Push(data []byte) {
	e.stack[e.stackTop] = data
	e.stackTop++
}

func (e *Engine) PushReg(reg byte) {
	e.Push(e.registers[reg])
}

func (e *Engine) Pop() {
	if e.stackTop == 0 {
		panic("Pop: stack is empty")
	}
	e.stackTop--
	e.stack[e.stackTop] = nil // for GC
}

func (e *Engine) PushBool(yn bool) {
	if yn {
		e.Push([]byte{0xFF})
	} else {
		e.Push(nil)
	}
}

func (e *Engine) IsFalse() bool {
	return len(e.Top()) == 0
}

func (e *Engine) Top() []byte {
	if e.stackTop == 0 {
		panic("Pop: stack is empty")
	}
	return e.stack[e.stackTop-1]
}

func (e *Engine) Jump8(offset byte) bool {
	e.registers[RegRemainingCode] = e.registers[RegRemainingCode][offset:]
	return true
}

func (e *Engine) Jump16(offset uint16) bool {
	e.registers[RegRemainingCode] = e.registers[RegRemainingCode][offset:]
	return true
}

func (e *Engine) run1Cycle() bool {
	var instrRunner InstructionRunner
	var paramData []byte
	instrRunner, paramData = e.opcodes.ParseInstruction(e.registers[RegRemainingCode])
	// TODO negerai
	e.registers[RegRemainingCode] = e.registers[RegRemainingCode][len(paramData):]
	return instrRunner(e, paramData)
}
