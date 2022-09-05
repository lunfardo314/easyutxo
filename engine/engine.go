package engine

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/lazyslice"
)

const (
	NumRegisters = 256
	MaxStack     = 100
)

type (
	Engine struct {
		opcodes      Opcodes
		code         []byte
		instrCounter int
		registers    [NumRegisters][]byte
		stack        [MaxStack][]byte
		stackTop     int
		ctx          *lazyslice.Tree
		exit         bool
	}
	OpCode interface {
		Bytes() []byte
		Uint16() uint16
		String() string
		Name() string
	}
	Opcodes interface {
		// ParseInstruction returns instruction runner, opcode length and instruction parameters
		ParseInstruction(code []byte) (InstructionRunner, []byte)
		ValidateOpcode(oc OpCode) error
	}
	InstructionParameterParser func(codeAfterOpcode []byte) (InstructionRunner, []byte)
	InstructionRunner          func(e *Engine, paramBytes []byte)
)

const (
	RegInvocationPath = byte(iota)
	RegInvocationData
	FirstWriteableRegister
)

// Run executes the script. If it returns, script is successful.
// If it panics, ledger is invalid
// invocationFullPath starts from validation root:
//  (a) (inputs, idx1, idx0, idxInsideOutput)
//  (b) (tx, context, idx, idxInsideOutput)
func Run(opcodes Opcodes, ctx *lazyslice.Tree, invocationFullPath lazyslice.TreePath, code, data []byte) {
	e := Engine{
		code:    code,
		opcodes: opcodes,
		ctx:     ctx,
	}
	e.registers[RegInvocationPath] = invocationFullPath
	e.registers[RegInvocationData] = data
	for e.run1Cycle() {
	}
}

func (e *Engine) Exit() {
	e.exit = true
}

func (e *Engine) RegValue(reg byte) []byte {
	return e.registers[reg]
}

func (e *Engine) Push(data []byte) {
	e.stack[e.stackTop] = data
	e.stackTop++
}

func (e *Engine) PushFromReg(reg byte) {
	e.Push(e.registers[reg])
}

func (e *Engine) PutToReg(reg byte, data []byte) {
	if reg < FirstWriteableRegister {
		panic(fmt.Errorf("attept to write to read-only register #%d", reg))
	}
	e.registers[reg] = data
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

func (e *Engine) Move(offset int) {
	e.instrCounter += offset
}

func (e *Engine) run1Cycle() bool {
	instrRunner, paramData := e.opcodes.ParseInstruction(e.code[e.instrCounter:])
	instrRunner(e, paramData)
	return !e.exit
}
