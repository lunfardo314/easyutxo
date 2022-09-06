package opcodes

import (
	"bytes"
	"fmt"

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
	OpsSigLockED25519
	OpsBlake2b
)

const (
	OplReserved126 = OpCode(iota + MaxShortOpcode + 1)
)

var All = allOpcodes{
	OpsNOP:            {"OpsNOP", noParamParser(runNOP)},
	OpsExit:           {"OpsExit", noParamParser(runExit)},
	OpsPop:            {"OpsPop", noParamParser(runPop)},
	OpsEqualLenShort:  {"OpsEqualLenShort", oneByteParameterParser(runEqualLenShort)},
	OpsEqualStackTop:  {"OpsEqualStackTop", noParamParser(runEqualStackTop)},
	OpsPushFromReg:    {"OpsPushFromReg", oneByteParameterParser(runPushFromReg)},
	OpsSaveParamToReg: {"OpsSaveParamToReg", saveParamToRegisterParser()},
	OpsPushFalse:      {"OpsPushFalse", noParamParser(runPushFalse)},
	// tree path/data manipulation
	OpsPushBytesFromPath:           {"OpsPushBytesFromPath", oneByteParameterParser(runOpsPushBytesFromPath)},
	OpsPushBytesFromPathAndIndex:   {"OpsPushBytesFromPathAndIndex", twoByteParameterParser(runOpsLoadBytesFromPathAndIndex)},
	OpsMakeUnlockBlockPathToReg:    {"OpsMakeUnlockBlockPathToReg", oneByteParameterParser(runMakeUnlockBlockPathToReg)},
	OpsPushTransactionEssenceBytes: {"OpsPushTransactionEssenceBytes", noParamParser(runPushTransactionEssenceBytes)},
	// flow control
	OpsJumpShortOnInputContext: {"OpsJumpShortOnInputContext", oneByteParameterParser(runJumpShortOnInputContext)},
	OpsJumpLongOnInputContext:  {"OpsJumpLongOnInputContext", twoByteParameterParser(runJumpLongOnInputContext)},
	OpsJumpShortOnTrue:         {"OpsJumpShortOnTrue", oneByteParameterParser(runJumpShortOnTrue)},
	OpsJumpLongOnTrue:          {"OpsJumpLongOnTrue", twoByteParameterParser(runJumpLongOnTrue)},
	OpsJumpShortOnFalse:        {"OpsJumpShortOnFalse", oneByteParameterParser(runJumpShortOnFalse)},
	OpsJumpLongOnFalse:         {"OpsJumpLongOnFalse", twoByteParameterParser(runJumpLongOnFalse)},
	// other
	OpsSigLockED25519: {"OpsSigLockED25519", noParamParser(runSigLockED25519)},
	OpsBlake2b:        {"OpsBlake2b", noParamParser(runBlake2b)},
	OplReserved126:    {"reserved long opcode", noParamParser(runReservedOpcode)},
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

func saveParamToRegisterParser() engine.InstructionParameterParser {
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
