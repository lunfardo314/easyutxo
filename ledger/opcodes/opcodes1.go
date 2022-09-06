package opcodes

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/lunfardo314/easyutxo/engine"
)

type opcodeDescriptor1 struct {
	symName      string
	description  string
	paramPattern string
	paramNames   string
	run          engine.InstructionRunner
}

type (
	opcodeDescriptorCompiled struct {
		opcode      OpCode
		run         engine.InstructionRunner
		description string
		params      []paramPatternCompiled
	}
	paramPatternCompiled struct {
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

var all = map[OpCode]*opcodeDescriptor1{
	OpsNOP:            {"nop", "no operation", "", "", runNOP},
	OpsExit:           {"exit", "exit script", "", "", runExit},
	OpsPop:            {"pop", "pop stack", "", "", runPop},
	OpsEqualLenShort:  {"==len", "length of register value is equal to parameter", "S", "register#-with-value", runEqualLenShort},
	OpsEqualStackTop:  {"==", "2 top stack values equal", "", "", runEqualStackTop},
	OpsPushFromReg:    {"pushReg", "push value from register", "S", "register#-with-value", runPushFromReg},
	OpsSaveParamToReg: {"saveToReg", "save parameter to register", "S,V", "register#,var-value", runSaveToRegister},
	OpsPushFalse:      {"false", "push false", "", "", runPushFalse},
	// tree path/data manipulation
	OpsPushBytesFromPath:           {"pushFromPath", "push value from path", "S", "register#-with-path", runOpsPushBytesFromPath},
	OpsPushBytesFromPathAndIndex:   {"pushFromPathIndex", "push value from path and index", "S", "register#-with-path", runOpsLoadBytesFromPathAndIndex},
	OpsMakeUnlockBlockPathToReg:    {"unlockBlockPath", "make and save unlock-block path to register", "S", "register#", runMakeUnlockBlockPathToReg},
	OpsPushTransactionEssenceBytes: {"pushTxEssence", "push transaction essence bytes", "", "", runPushTransactionEssenceBytes},
	// flow control
	OpsJumpShortOnInputContext: {"ifInputContext->", "jump short if invocation is input context", "JS", "target-short", runJumpShortOnInputContext},
	OpsJumpLongOnInputContext:  {"ifInputContext>>>", "jump long if invocation is input context", "JL", "target-long", runJumpLongOnInputContext},
	OpsJumpShortOnTrue:         {"ifTrue->", "jump short if stack top is true", "JS", "target-short", runJumpShortOnTrue},
	OpsJumpLongOnTrue:          {"ifTrue>>>", "jump long if stack top is true", "JL", "target-long", runJumpLongOnTrue},
	OpsJumpShortOnFalse:        {"ifFalse->", "jump short if stack top is false", "JS", "target-short", runJumpShortOnFalse},
	OpsJumpLongOnFalse:         {"ifFalse>>>", "jump long if stack top is false", "JL", "target-long", runJumpLongOnFalse},
	// other
	OpsVerifySigED25519: {"verifySigED25519", "verify ED25519 signature", "", "", runSigLockED25519},
	OpsBlake2b:          {"blake2b", "hash blake2b", "", "", runBlake2b},
	OplReserved126:      {"reserved126", "fake opcode", "", "", runReservedOpcode},
}

var All1 = mustPreCompileOpcodes(all)

func throwErr(format string, args ...interface{}) {
	panic(fmt.Errorf("pre-compile error: "+format+"\n", args...))
}

func mustPreCompileOpcodes(ocRaw map[OpCode]*opcodeDescriptor1) map[string]*opcodeDescriptorCompiled {
	ret := make(map[string]*opcodeDescriptorCompiled)
	for oc, dscr := range ocRaw {
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
		if _, already := ret[dscr.symName]; already {
			throwErr("repeating opcode name: '%s' (%s)", dscr.symName, dscr.description)
		}
		ret[dscr.symName] = &opcodeDescriptorCompiled{
			opcode:      oc,
			run:         dscr.run,
			description: dscr.description,
			params:      make([]paramPatternCompiled, parNum),
		}
		for i := range ret[dscr.symName].params {
			ret[dscr.symName].params[i].paramName = strings.TrimSpace(splitN[i])
			ret[dscr.symName].params[i].paramType = splitP[i]
			if len(ret[dscr.symName].params[i].paramName) == 0 {
				throwErr("opcode parameter name can't be empty @ '%s' (%s)", dscr.symName, dscr.description)
			}
			switch ret[dscr.symName].params[i].paramType {
			case paramType8, paramType16, paramTypeVariable, paramTypeShortTarget, paramTypeLongTarget:
			default:
				throwErr("wrong parameter pattern '%s' @ '%s' (%s)", splitP[i], dscr.symName, dscr.description)
			}
		}
	}
	return ret
}

func CompileSource(sourceCode string) ([]byte, error) {
	lines := splitLines(sourceCode)
	for lineno, line := range lines {
		instr, _, _ := strings.Cut(line, ";")
		l := strings.TrimSpace(instr)
		if len(l) == 0 {
			continue
		}
		instr = strings.TrimSpace(instr)
		if strings.HasPrefix(instr, ">") {
			instr = strings.TrimPrefix(instr, ">")
			instr = strings.TrimSpace(instr)
			fmt.Printf("%2d: label: '%s'\n", lineno, instr)
		} else {
			opcode, params, _ := strings.Cut(instr, " ")
			opcode = strings.TrimSpace(opcode)
			params = strings.TrimSpace(params)
			par := strings.Split(params, ",")
			fmt.Printf("%2d: opcode: '%s', params: %v\n", lineno, opcode, par)
		}
	}
	return nil, nil
}

func splitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}
