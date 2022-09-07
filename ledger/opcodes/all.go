package opcodes

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/ledger/path"
)

const (
	OpsNOP = OpCode(iota)
	OpsExit
	OpsPop
	OpsEqualLenShort
	OpsEqualStackTop
	OpsPushFromReg
	OpsSaveParamToReg
	OpsPushFalse
	// tree path/data manipulation
	OpsPushBytesFromPath
	OpsPushBytesFromPathAndIndex
	OpsPushTransactionEssenceBytes

	OpsMakeUnlockBlockPathToReg
	// control
	OpsJumpShortOnInputContext
	OpsJumpLongOnInputContext
	OpsJumpShortOnTrue
	OpsJumpLongOnTrue
	OpsJumpShortOnFalse
	OpsJumpLongOnFalse
	// other
	OpsVerifySigED25519
	OpsBlake2b
)

const (
	OplReserved126 = OpCode(iota + MaxShortOpcode + 1)
)

var allRaw = map[OpCode]*opcodeDescriptor{
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

var All, allLookup = mustPreCompileOpcodes(allRaw)

func throwErr(format string, args ...interface{}) {
	panic(fmt.Errorf("pre-compile error: "+format+"\n", args...))
}

func mustPreCompileOpcodes(ocRaw map[OpCode]*opcodeDescriptor) (allOpcodesPreCompiled, map[string]*opcodeDescriptorCompiled) {
	retPrecompiled := make(allOpcodesPreCompiled)
	retLookup := make(map[string]*opcodeDescriptorCompiled)
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
		if _, already := retLookup[dscr.symName]; already {
			throwErr("repeating opcode name: '%s' (%s)", dscr.symName, dscr.description)
		}

		retPrecompiled[oc] = &opcodeDescriptorCompiled{
			dscr:   dscr,
			params: make([]paramsTemplateCompiled, parNum),
		}
		retLookup[dscr.symName] = retPrecompiled[oc]

		for i := range retLookup[dscr.symName].params {
			retLookup[dscr.symName].params[i].paramName = strings.TrimSpace(splitN[i])
			retLookup[dscr.symName].params[i].paramType = splitP[i]
			if len(retLookup[dscr.symName].params[i].paramName) == 0 {
				throwErr("opcode parameter name can't be empty @ '%s' (%s)", dscr.symName, dscr.description)
			}
			switch retLookup[dscr.symName].params[i].paramType {
			case paramType8, paramType16, paramTypeVariable, paramTypeShortTarget, paramTypeLongTarget:
			default:
				throwErr("wrong parameter pattern '%s' @ '%s' (%s)", splitP[i], dscr.symName, dscr.description)
			}
		}
	}
	return retPrecompiled, retLookup
}

func mustParLen(par []byte, n int) {
	if len(par) != n {
		panic(fmt.Errorf("instruction parameter must be %d bytes long", n))
	}
}

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

// runEqualLenShort pushes true/false if length of data at the stack top equals to the byte parameter of the instruction
// Leaves data on the stack
func runEqualLenShort(e *engine.Engine, d []byte) {
	mustParLen(d, 1)
	e.PushBool(len(e.Top()) == int(d[0]))
	e.Move(1 + 1)
}

// runEqualStackTop compares two top stack elements and removes thjem
func runEqualStackTop(e *engine.Engine, d []byte) {
	mustParLen(d, 0)
	e.PushBool(bytes.Equal(e.Pop(), e.Pop()))
	e.Move(1)
}

func runPushFromReg(e *engine.Engine, d []byte) {
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

func runMakeUnlockBlockPathToReg(e *engine.Engine, d []byte) {
	mustParLen(d, 1)
	unlockBlockPath := path.UnlockBlockPathFromInputInvocationPath(e.RegValue(engine.RegInvocationPath))
	e.PutToReg(d[0], unlockBlockPath)
	e.Move(1 + 1)
}

func runPushTransactionEssenceBytes(e *engine.Engine, d []byte) {
	mustParLen(d, 0)
	var buf bytes.Buffer

	buf.Write(e.BytesAtPath(path.GlobalInputIDsLong))
	buf.Write(e.BytesAtPath(path.GlobalOutputGroups))
	buf.Write(e.BytesAtPath(path.GlobalTimestamp))
	buf.Write(e.BytesAtPath(path.GlobalContextCommitment))
	e.Push(buf.Bytes())
	e.Move(1)
}

func runOpsPushBytesFromPath(e *engine.Engine, d []byte) {
	mustParLen(d, 1)
	e.Push(e.BytesAtPath(e.RegValue(d[0])))
}

func runOpsLoadBytesFromPathAndIndex(e *engine.Engine, d []byte) {
	mustParLen(d, 2)
	e.Push(e.GetDataAtIdx(d[1], e.RegValue(d[0])))
	e.Move(1 + 1)
}
