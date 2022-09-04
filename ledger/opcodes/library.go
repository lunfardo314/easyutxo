package opcodes

import (
	"github.com/lunfardo314/easyutxo/engine"
)

const (
	OPS_EXIT = OpCode(iota)
	OPS_SIG_ED25519
)

const (
	OPL_L1 = OpCode(iota + MaxShortOpcode + 1)
)

var Library = library{
	OPS_EXIT:        {"OPS_EXIT", op1ByteParser(opExitRunner)},
	OPS_SIG_ED25519: {"OPS_SIG_ED25519", op1ByteParser(opSigED25519Runner)},
	OPL_L1:          {"mocked long opcode", nil},
}

func op1ByteParser(runner engine.InstructionRunner) engine.InstructionParser {
	return func(codeAfterOpcode []byte) (engine.InstructionRunner, []byte) {
		return runner, codeAfterOpcode
	}
}

func opExitRunner(_ engine.Engine) bool {
	return false
}

func opSigED25519Runner(_ engine.Engine) bool {
	return false
}
