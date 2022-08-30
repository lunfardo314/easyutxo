package engine

import (
	"fmt"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

type OpCode uint16

const extendedOpcodeMask = 0x80

func isShortOpcode(firstByte byte) bool {
	return firstByte&extendedOpcodeMask == 0
}

func (c OpCode) AsBytes() []byte {
	ret := easyutxo.EncodeInteger(uint16(c))
	if isShortOpcode(ret[1]) {
		ret = ret[:1]
	}
	return ret
}

func (c OpCode) Name() string {
	if dscr, ok := opcodes[c]; ok {
		return dscr.name
	}
	return "(wrong opcode)"
}

func (c OpCode) String() string {
	return fmt.Sprintf("%s(0x%02X)", c.Name(), uint16(c))
}

type instructionParser func(codeAfterOpcode []byte) (instructionRunner, []byte)
type instructionRunner func(ctx *lazyslice.Tree) bool

func parseOpcode(code []byte) (OpCode, []byte) {
	var opcode OpCode
	var parOffset int
	if isShortOpcode(code[0]) {
		opcode = OpCode(easyutxo.DecodeInteger[uint16]([]byte{0, code[0]}))
		parOffset = 1
	} else {
		if len(code) < 2 {
			panic("unexpected end of the remainingCode")
		}
		opcode = OpCode(easyutxo.DecodeInteger[uint16](code[:2]))
		parOffset = 2
	}
	return opcode, code[parOffset:]
}

// parseInstruction return first parsed instruction and remaining remainingCode
func parseInstruction(code []byte) (instructionRunner, []byte) {
	if len(code) == 0 {
		return opExitRunner, code
	}
	opcode, codeAfterOpcode := parseOpcode(code)
	dscr, found := opcodes[opcode]
	if !found {
		panic(opcode)
	}
	return dscr.parser(codeAfterOpcode)
}

const (
	OP_EXIT = OpCode(iota)
)

type opcodeDescriptor struct {
	name   string
	parser instructionParser
}

var opcodes = map[OpCode]opcodeDescriptor{
	OP_EXIT: {"OP_EXIT", opExitParser},
}

func opExitParser(codeAfterOpcode []byte) (instructionRunner, []byte) {
	return opExitRunner, codeAfterOpcode
}

func opExitRunner(ctx *lazyslice.Tree) bool {
	return false
}
