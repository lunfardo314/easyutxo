package opcodes

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/engine"
)

const (
	OPS_NOP = OpCode(iota)
	OPS_EXIT
	OPS_POP
	OPS_CHECK_LEN8
	OPS_PUSH_REG
	// control
	OPS_JUMP8_ON_TRUE
	OPS_JUMP8_ON_FALSE
	// other
	OPS_SIGLOCK_ED25519
)

const (
	OPL_RESERVED126 = OpCode(iota + MaxShortOpcode + 1)
)

var All = allOpcodes{
	OPS_NOP:             {"OPS_NOP", noParamParser(nopRunner)},
	OPS_EXIT:            {"OPS_EXIT", noParamParser(exitRunner)},
	OPS_POP:             {"OPS_POP", noParamParser(popRunner)},
	OPS_CHECK_LEN8:      {"OPS_CHECK_LEN8", oneByteParameterParser(checkLen8Runner)},
	OPS_PUSH_REG:        {"OPS_PUSH_REG", oneByteParameterParser(pushRegRunner)},
	OPS_SIGLOCK_ED25519: {"OPS_SIGLOCK_ED25519", noParamParser(opSigED25519Runner)},
	OPL_RESERVED126:     {"reserved long opcode", noParamParser(reservedOpcodeRunner)},
}

func mustParLen(par []byte, n int) {
	if len(par) != n {
		panic(fmt.Errorf("instruction parameter must be #%d bytes long", n))
	}
}

func noParamParser(runner engine.InstructionRunner) engine.InstructionParser {
	return func(codeAfterOpcode []byte) (engine.InstructionRunner, []byte) {
		return runner, nil
	}
}

func oneByteParameterParser(runner engine.InstructionRunner) engine.InstructionParser {
	return func(codeAfterOpcode []byte) (engine.InstructionRunner, []byte) {
		return runner, []byte{codeAfterOpcode[0]}
	}
}

func nopRunner(_ *engine.Engine, d []byte) bool {
	mustParLen(d, 0)
	return true
}

func exitRunner(_ *engine.Engine, d []byte) bool {
	mustParLen(d, 0)
	return false
}

func reservedOpcodeRunner(_ *engine.Engine, _ []byte) bool {
	panic("reserved opcode")
}

func popRunner(e *engine.Engine, d []byte) bool {
	mustParLen(d, 0)
	e.Pop()
	return false
}

// checkLen8Runner pushes true/false if length of data at the stack top  equals to the byte parameter of the instruction
func checkLen8Runner(e *engine.Engine, d []byte) bool {
	mustParLen(d, 1)
	e.PushBool(len(e.Top()) == int(d[0]))
	return true
}

func pushRegRunner(e *engine.Engine, d []byte) bool {
	mustParLen(d, 1)
	e.PushReg(d[0])
	return true
}
