package opcodes

import (
	"bytes"
	"fmt"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/ledger/path"
)

var allRaw1Byte = []*opcodeDescriptor{
	{"nop", "no operation", "", "", runNOP},
	{"exit", "exit script", "", "", runExit},
	{"pop", "pop stack", "", "", runPop},
	{"<-reg", "push value from register to stack", "S", "register#-with-value", runPushFromReg},
	{"reg<-", "save parameter to register", "S,V", "register#,var-value", runSaveToRegister},
	{"<-", "push parameter to stack", "V", "var-value", runPushParameterToStack},
	{"<-size", "push 2 bytes uint16 size of value at top", "", "", runSize},
	{"==", "2 top stack values equal", "", "", runEqualStackTop},

	// --------------------------------------------------------
	// tree path/data manipulation
	{"pushFromPath", "push value from path", "S", "register#-with-path", runOpsPushBytesFromPath},
	{"pushFromPathIndex", "push value from path and index", "S,S", "register#-with-path, element_index", runOpsLoadBytesFromPathAndIndex},
	{"makeUnlockBlockPath", "make and save unlock-block path to register", "S", "register#", runMakeUnlockBlockPathToReg},
	{"pushTxEssence", "push transaction essence bytes", "", "", runPushTransactionEssenceBytes},
	// flow control
	{"ifInputContext>", "jump short if invocation is input context", "JS", "target-short", runJumpShortOnInputContext},
	{"ifInputContext>>>", "jump long if invocation is input context", "JL", "target-long", runJumpLongOnInputContext},
	{"ifTrue>", "jump short if stack top is true", "JS", "target-short", runJumpShortOnTrue},
	{"ifTrue>>>", "jump long if stack top is true", "JL", "target-long", runJumpLongOnTrue},
	{"ifFalse>", "jump short if stack top is false", "JS", "target-short", runJumpShortOnFalse},
	{"ifFalse>>>", "jump long if stack top is false", "JL", "target-long", runJumpLongOnFalse},
	// other
	{"verifySigED25519", "verify ED25519 signature", "", "", runSigLockED25519},
	{"blake2b", "hash blake2b", "", "", runBlake2b},
}

var allRaw12Byte = []*opcodeDescriptor{
	{"reserved126", "fake opcode", "", "", runReservedOpcode},
}

var All, allSymLookup = mustPreCompileOpcodes(allRaw1Byte, allRaw12Byte)

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

func runPushParameterToStack(e *engine.Engine, d []byte) {
	if len(d) < 1 {
		panic("instruction parameter expected to be at least 2 bytes long")
	}
	mustParLen(d[1:], int(d[0]))
	e.Push(d[1:])
	e.Move(1 + int(d[1]))
}

func runSize(e *engine.Engine, d []byte) {
	mustParLen(d, 0)
	e.Push(easyutxo.EncodeInteger(uint16(len(e.Top()))))
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
