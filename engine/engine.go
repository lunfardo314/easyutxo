package engine

import (
	"fmt"

	"github.com/lunfardo314/easyutxo"
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
		currentPos   int
		instrCounter int
		registers    [NumRegisters][]byte
		stack        [MaxStack][]byte
		stackTop     int
		ctx          *lazyslice.Tree
		trace        bool
		exit         bool
	}
	OpCode interface {
		Bytes() []byte
		Uint16() uint16
		String() string
		Name() string
		Valid() bool
	}
	Opcodes interface {
		// ParseInstruction returns instruction runner, opcode length and instruction parameters
		ParseInstruction(code []byte) (InstructionRunner, []byte)
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
func Run(opcodes Opcodes, ctx *lazyslice.Tree, invocationFullPath lazyslice.TreePath, code, data []byte, trace ...bool) {
	e := Engine{
		code:    code,
		opcodes: opcodes,
		ctx:     ctx,
	}
	if len(trace) > 0 {
		e.trace = trace[0]
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

func (e *Engine) Pop() []byte {
	if e.stackTop == 0 {
		panic("Pop: stack is empty")
	}
	e.stackTop--
	ret := e.stack[e.stackTop]
	e.stack[e.stackTop] = nil // for GC
	return ret
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
	e.currentPos += offset
}

func (e *Engine) BytesAtPath(p lazyslice.TreePath) []byte {
	return e.ctx.BytesAtPath(p)
}

func (e *Engine) GetDataAtIdx(idx byte, p lazyslice.TreePath) []byte {
	return e.ctx.GetDataAtIdx(idx, p)
}

func (e *Engine) traceString() string {
	if !e.trace {
		return "(no trace available)"
	}
	return "(tracing not implemented)"
}

func (e *Engine) run1Cycle() bool {
	var instrRunner InstructionRunner
	var paramData []byte
	err := easyutxo.CatchPanic(func() {
		instrRunner, paramData = e.opcodes.ParseInstruction(e.code[e.currentPos:])
	})
	if err != nil {
		panic(fmt.Errorf("cannot parse instruction after instuction %d @ script position %d. Trace:\n%s",
			e.instrCounter, e.currentPos, e.traceString()))
	}
	err = easyutxo.CatchPanic(func() {
		instrRunner(e, paramData)
	})
	e.instrCounter++
	return !e.exit
}
