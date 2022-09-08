package opcodes

import (
	"fmt"
	"strings"

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
		opcode OpCode
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

func (all allOpcodesPreCompiled) ParseInstruction(code []byte) (engine.InstructionRunner, [][]byte) {
	if len(code) == 0 {
		return runExit, nil
	}
	opcode, offs := ParseOpcode(code)
	dscr, found := all[opcode]
	if !found {
		panic(opcode)
	}
	return dscr.dscr.run, parseParams(code[offs:], dscr.params)
}

func parseParams(code []byte, templates []paramsTemplateCompiled) [][]byte {
	ret := make([][]byte, 0)
	offs := 0
	for _, t := range templates {
		switch t.paramType {
		case paramType8:
			ret = append(ret, code[offs:offs+1])
			offs += 1
		case paramType16:
			ret = append(ret, code[offs:offs+2])
			offs += 2
		case paramTypeVariable:
			ret = append(ret, code[offs+1:offs+int(code[offs])+1])
			offs += int(code[offs]) + 1
		case paramTypeShortTarget:
			ret = append(ret, code[offs:offs+1])
			offs++
		case paramTypeLongTarget:
			ret = append(ret, code[offs:offs+2])
			offs += 2
		default:
			panic("parseParams: wrong param template")
		}
	}
	return ret
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
	if dscr, ok := All[c]; ok {
		return dscr.dscr.symName
	}
	return "(wrong OpCode)"
}

func (c OpCode) Uint16() uint16 {
	return uint16(c)
}

func (c OpCode) Valid() bool {
	_, found := All[c]
	return found
}

func (c OpCode) IsShort() bool {
	return uint16(c) <= MaxShortOpcode
}

func throwErr(format string, args ...interface{}) {
	panic(fmt.Errorf("pre-compile error: "+format+"\n", args...))
}

func mustPreCompileOpcodes(ocRaw1Byte, ocRaw2Byte []*opcodeDescriptor) (allOpcodesPreCompiled, map[string]*opcodeDescriptorCompiled) {
	if len(ocRaw1Byte) > 127 {
		panic("can't be more than 127 one-byte opcodes")
	}
	retPrecompiled := make(allOpcodesPreCompiled)
	retSymLookup := make(map[string]*opcodeDescriptorCompiled)
	// for both table do the same
	for tableNo, table := range [][]*opcodeDescriptor{ocRaw1Byte, ocRaw2Byte} {
		for oc, dscr := range table {
			if dscr == nil {
				continue // reserved
			}
			trimmed := strings.TrimSpace(dscr.paramPattern)
			parNum := 0
			var splitN, splitP []string
			if len(trimmed) > 0 {
				splitP = strings.Split(dscr.paramPattern, ",")
				splitN = strings.Split(dscr.paramNames, ",")
				if len(splitP) != len(splitN) {
					throwErr("number of parameter patterns not equal to number of parameter names @ '%s' (%s)", dscr.symName, dscr.description)
				}
				parNum = len(splitP)
			}
			if _, already := retSymLookup[dscr.symName]; already {
				throwErr("repeating opcode name: '%s' (%s)", dscr.symName, dscr.description)
			}

			opcode := OpCode(oc + 127*tableNo)
			retPrecompiled[opcode] = &opcodeDescriptorCompiled{
				opcode: opcode,
				dscr:   dscr,
				params: make([]paramsTemplateCompiled, parNum),
			}
			retSymLookup[dscr.symName] = retPrecompiled[opcode]

			for i := range retSymLookup[dscr.symName].params {
				retSymLookup[dscr.symName].params[i].paramName = strings.TrimSpace(splitN[i])
				retSymLookup[dscr.symName].params[i].paramType = splitP[i]
				if len(retSymLookup[dscr.symName].params[i].paramName) == 0 {
					throwErr("opcode parameter name can't be empty @ '%s' (%s)", dscr.symName, dscr.description)
				}
				switch retSymLookup[dscr.symName].params[i].paramType {
				case paramType8, paramType16, paramTypeVariable, paramTypeShortTarget, paramTypeLongTarget:
				default:
					throwErr("wrong parameter pattern '%s' @ '%s' (%s)", splitP[i], dscr.symName, dscr.description)
				}
			}
		}

	}
	return retPrecompiled, retSymLookup
}
