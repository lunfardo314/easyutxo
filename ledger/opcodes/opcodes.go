package opcodes

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/engine"
)

type (
	OpCode           uint16
	opcodeDescriptor struct {
		name   string
		parser engine.InstructionParser
	}
	library map[OpCode]opcodeDescriptor
)

func (lib library) ParseInstruction(code []byte) (engine.InstructionRunner, []byte) {
	if len(code) == 0 {
		return opExitRunner, code
	}
	opcode, codeAfterOpcode := ParseOpcode(code)
	dscr, found := lib[opcode]
	if !found {
		panic(opcode)
	}
	return dscr.parser(codeAfterOpcode)
}

const (
	MaxShortOpcode = uint16(^byte(0x80)) // uint16(127)
)

func isShortOpcodeByte(firstByte byte) bool {
	return uint16(firstByte) <= MaxShortOpcode
}

func (c OpCode) Bytes() []byte {
	if c.IsShort() {
		return []byte{byte(c)}
	}
	return []byte{byte(c), byte(c >> 8)}
}

func (c OpCode) String() string {
	name := c.Name()
	return fmt.Sprintf("%s(%d)", name, c)
}

func (c OpCode) Name() string {
	if dscr, ok := Library[c]; ok {
		return dscr.name
	}
	return "(wrong OpCode)"
}

func (c OpCode) Uint16() uint16 {
	return uint16(c)
}

func (c OpCode) IsShort() bool {
	return uint16(c) <= MaxShortOpcode
}

func ParseOpcode(code []byte) (OpCode, []byte) {
	var op OpCode
	var retOffset int
	if isShortOpcodeByte(code[0]) {
		op = OpCode(code[0])
		retOffset = 1
	} else {
		if len(code) < 2 {
			panic("unexpected end of the code")
		}
		op = OpCode(uint16(code[0]) + uint16(code[1])<<8)
		retOffset = 2
	}
	return op, code[retOffset:]
}
