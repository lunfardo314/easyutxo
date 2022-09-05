package opcodes

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/engine"
)

const (
	OPS_NOP = OpCode(iota)
	OPS_EXIT
	OPS_POP
	OPS_EQUAL_LEN8
	OPS_LOAD_FROM_REG
	OPS_SAVE_TO_REG
	OPS_PUSH_FALSE
	// control
	OPS_JUMP8_ON_INPUT_CTX
	OPS_JUMP16_ON_INPUT_CTX
	OPS_JUMP8_ON_TRUE
	OPS_JUMP16_ON_TRUE
	OPS_JUMP8_ON_FALSE
	OPS_JUMP16_ON_FALSE
	// other
	OPS_SIGLOCK_ED25519
)

const (
	OPL_RESERVED126 = OpCode(iota + MaxShortOpcode + 1)
)

var All = allOpcodes{
	OPS_NOP:           {"OPS_NOP", noParamParser(runNOP)},
	OPS_EXIT:          {"OPS_EXIT", noParamParser(runExit)},
	OPS_POP:           {"OPS_POP", noParamParser(runPop)},
	OPS_EQUAL_LEN8:    {"OPS_EQUAL_LEN8", oneByteParameterParser(runEqual8)},
	OPS_LOAD_FROM_REG: {"OPS_LOAD_FROM_REG", oneByteParameterParser(runLoadFromReg)},
	OPS_SAVE_TO_REG:   {"OPS_SAVE_TO_REG", saveToRegisterParser()},
	OPS_PUSH_FALSE:    {"OPS_PUSH_FALSE", noParamParser(runPushFalse)},
	// flow control
	OPS_JUMP8_ON_INPUT_CTX:  {"OPS_JUMP8_ON_INPUT_CTX", oneByteParameterParser(runJump8OnInputContext)},
	OPS_JUMP16_ON_INPUT_CTX: {"OPS_JUMP16_ON_INPUT_CTX", twoByteParameterParser(runJump16OnInputContext)},
	OPS_JUMP8_ON_TRUE:       {"OPS_JUMP8_ON_TRUE", oneByteParameterParser(runJump8OnTrue)},
	OPS_JUMP16_ON_TRUE:      {"OPS_JUMP16_ON_TRUE", twoByteParameterParser(runJump16OnTrue)},
	OPS_JUMP8_ON_FALSE:      {"OPS_JUMP8_ON_FALSE", oneByteParameterParser(runJump8OnFalse)},
	OPS_JUMP16_ON_FALSE:     {"OPS_JUMP16_ON_FALSE", twoByteParameterParser(runJump16OnFalse)},
	// other
	OPS_SIGLOCK_ED25519: {"OPS_SIGLOCK_ED25519", noParamParser(opSigED25519Runner)},
	OPL_RESERVED126:     {"reserved long opcode", noParamParser(runReservedOpcode)},
}

func mustParLen(par []byte, n int) {
	if len(par) != n {
		panic(fmt.Errorf("instruction parameter must be %d bytes long", n))
	}
}

// parsers

func noParamParser(runner engine.InstructionRunner) engine.InstructionParameterParser {
	return func(codeAfterOpcode []byte) (engine.InstructionRunner, []byte) {
		return runner, nil
	}
}

func oneByteParameterParser(runner engine.InstructionRunner) engine.InstructionParameterParser {
	return func(codeAfterOpcode []byte) (engine.InstructionRunner, []byte) {
		return runner, codeAfterOpcode[:1]
	}
}

func twoByteParameterParser(runner engine.InstructionRunner) engine.InstructionParameterParser {
	return func(codeAfterOpcode []byte) (engine.InstructionRunner, []byte) {
		return runner, codeAfterOpcode[:2]
	}
}

func saveToRegisterParser() engine.InstructionParameterParser {
	return func(codeAfterOpcode []byte) (engine.InstructionRunner, []byte) {
		return runSaveToRegister, codeAfterOpcode[:2+codeAfterOpcode[1]]
	}
}

// runners

func runNOP(e *engine.Engine, d []byte) {
	mustParLen(d, 0)
	e.Move(1)
}

func runExit(e *engine.Engine, d []byte) {
	mustParLen(d, 0)
	e.Exit()
}

func runReservedOpcode(_ *engine.Engine, _ []byte) {
	panic("reserved opcode")
}

func runPop(e *engine.Engine, d []byte) {
	mustParLen(d, 0)
	e.Pop()
	e.Move(1)
}

// runEqual8 pushes true/false if length of data at the stack top equals to the byte parameter of the instruction
func runEqual8(e *engine.Engine, d []byte) {
	mustParLen(d, 1)
	e.PushBool(len(e.Top()) == int(d[0]))
	e.Move(1 + 1)
}

func runLoadFromReg(e *engine.Engine, d []byte) {
	mustParLen(d, 1)
	e.PushFromReg(d[0])
	e.Move(1 + 1)
}

func runSaveToRegister(e *engine.Engine, d []byte) {
	if len(d) < 2 {
		panic("instruction parameter expected to be at least 2 bytes long")
	}
	mustParLen(d[2:], int(d[1]))
	e.PutToReg(d[0], d[2:])
	e.Move(2 + int(d[1]))
}

func runPushFalse(e *engine.Engine, d []byte) {
	mustParLen(d, 0)
	e.PushBool(false)
	e.Move(1)
}
