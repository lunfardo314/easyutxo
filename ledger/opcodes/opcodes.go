package opcodes

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/engine"
)

type (
	OpCode           uint16
	opcodeDescriptor struct {
		symName      string
		description  string
		paramPattern string
		paramNames   string
		run          engine.InstructionRunner
	}
	opcodeDescriptorCompiled struct {
		dscr   *opcodeDescriptor
		params []paramsTemplateCompiled
	}
	allOpcodesPreCompiled  map[OpCode]*opcodeDescriptorCompiled
	paramsTemplateCompiled struct {
		paramType string
		paramName string
	}
)

const (
	paramType8           = "S"
	paramType16          = "L"
	paramTypeVariable    = "V"
	paramTypeShortTarget = "JS"
	paramTypeLongTarget  = "JL"
)

func (all allOpcodesPreCompiled) ParseInstruction(code []byte) (engine.InstructionRunner, []byte) {
	if len(code) == 0 {
		return runExit, code
	}
	opcode, offs := ParseOpcode(code)
	dscr, found := all[opcode]
	if !found {
		panic(opcode)
	}
	return dscr.dscr.run, parseParams(code[offs:], dscr.params)
}

func parseParams(code []byte, templates []paramsTemplateCompiled) []byte {
	offs := 0
	for _, t := range templates {
		switch t.paramType {
		case paramType8:
			offs++
		case paramType16:
			offs += 2
		case paramTypeVariable:
			offs += int(code[offs])
		case paramTypeShortTarget:
			offs++
		case paramTypeLongTarget:
			offs += 2
		default:
			panic("parseParams: wrong param template")
		}
	}
	return code[:offs]
}

func (all allOpcodesPreCompiled) ValidateOpcode(oc engine.OpCode) error {
	if _, found := all[oc.(OpCode)]; !found {
		return fmt.Errorf("wrong opcode %d", oc)
	}
	return nil
}

func ParseOpcode(code []byte) (OpCode, int) {
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
	return op, retOffset
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
	if dscr, ok := allRaw[c]; ok {
		return dscr.symName
	}
	return "(wrong OpCode)"
}

func (c OpCode) Uint16() uint16 {
	return uint16(c)
}

func (c OpCode) IsShort() bool {
	return uint16(c) <= MaxShortOpcode
}
