package opcodes

import (
	"bytes"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/ledger/globalpath"
)

var allRaw1Byte = []*opcodeDescriptor{
	{"nop", "no operation", "", "", runNOP},
	{"exit", "exit script", "", "", runExit},
	{"drop", "drop elements from stack", "S", "num-elements", runDrop},
	{"swap", "swaps 2 top elements of the stack", "", "", runSwap},
	{"reg->stack", "push value from register to stack", "S", "register#-with-value", runPushFromReg},
	{"param->reg", "save parameter to register", "S,V", "register#,var-value", runSaveParamToRegister},
	{"param->stack", "push parameter to stack", "V", "var-value", runPushParameterToStack},
	{"stack->reg", "save stack top to register", "S", "register#", runSaveStackToRegister},
	{"[:]->stack", "push slice of the top element", "S,S", "from_index,to_index", runSlice},
	{"size16->stack", "push 2 bytes uint16 size of value at top", "", "", runSize},
	{"==", "2 top stack values equal", "", "", runEqualStackTop},
	{"[:]==param", "compares slice of the stack top with param", "S,S,V", "from-idx,to-idx,const", runEqualSliceWithParam},
	{"concatReplace", "concatenate several elements and replace the top", "S", "S", runConcat},
	{"pushFromPathIndex", "push value from globalpath (top-1) and index (top)", "", "", runPushFromPathAndIndex},
	{"pushFromPathParam", "push bytes from global path", "V", "globalpath", runPushBytesFromPath},
	{"loadRefInputBlock", "load referenced block", "", "", runLoadRefInputBlock},

	// --------------------------------------------------------
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

func runNOP(e *engine.Engine, p [][]byte) {
	e.Move(1)
}

func runExit(e *engine.Engine, p [][]byte) {
	e.Exit()
}

func runReservedOpcode(_ *engine.Engine, _ [][]byte) {
	panic("reserved opcode")
}

func runDrop(e *engine.Engine, p [][]byte) {
	for i := 0; i < int(p[0][0]); i++ {
		e.Pop()
	}
	e.Move(1 + 1)
}

func runSwap(e *engine.Engine, _ [][]byte) {
	t0 := e.Pop()
	t1 := e.Pop()
	e.Push(t0)
	e.Push(t1)
	e.Move(1)
}

// runEqualStackTop compares two top stack elements
func runEqualStackTop(e *engine.Engine, p [][]byte) {
	e.PushBool(bytes.Equal(e.Top(), e.Top()))
	e.Move(1)
}

func runEqualSliceWithParam(e *engine.Engine, p [][]byte) {
	e.PushBool(bytes.Equal(e.Top()[p[0][0]:p[1][0]], p[2]))
	e.Move(1 + 2 + len(p[2]) + 1)
}

func runPushFromReg(e *engine.Engine, p [][]byte) {
	e.PushFromReg(p[0][0])
	e.Move(1 + 1)
}

func runSaveParamToRegister(e *engine.Engine, p [][]byte) {
	e.PutToReg(p[0][0], p[1])
	e.Move(1 + len(p[1]) + 1)
}

func runPushParameterToStack(e *engine.Engine, p [][]byte) {
	e.Push(p[0])
	e.Move(1 + len(p[0]) + 1)
}

func runSaveStackToRegister(e *engine.Engine, p [][]byte) {
	e.PutToReg(p[0][0], e.Top())
	e.Move(1 + 1)
}

func runSlice(e *engine.Engine, p [][]byte) {
	e.Push(e.Top()[p[0][0]:p[1][0]])
	e.Move(1 + 2)
}

func runConcat(e *engine.Engine, p [][]byte) {
	var buf bytes.Buffer
	for i := 0; i < int(p[0][0]); i++ {
		buf.Write(e.Pop())
	}
	e.Push(buf.Bytes())
	e.Move(1 + 1)
}

func runSize(e *engine.Engine, p [][]byte) {
	e.Push(easyutxo.EncodeInteger(uint16(len(e.Top()))))
	e.Move(1)
}

func runMakeUnlockBlockPathToReg(e *engine.Engine, p [][]byte) {
	unlockBlockPath := globalpath.UnlockBlockPathFromInputPath(e.RegValue(engine.RegInvocationPath))
	e.PutToReg(p[0][0], unlockBlockPath)
	e.Move(1 + 1)
}

func runPushBytesFromPath(e *engine.Engine, p [][]byte) {
	e.Push(e.BytesAtPath(e.RegValue(p[0][0])))
}

func runPushFromPathAndIndex(e *engine.Engine, _ [][]byte) {
	index := e.Pop()
	e.Push(e.GetDataAtIdx(index[0], e.Top()))
	e.Move(1)
}

// ref block is 2 bytes of long input index and 1 byte of block inside output
func runLoadRefInputBlock(e *engine.Engine, _ [][]byte) {
	idx := easyutxo.DecodeInteger[uint16](e.Top()[:2])

	e.Push(e.GetDataAtIdx(e.Top()[2], globalpath.ConsumedOutput(idx)))
	e.Move(1)
}
