package opcodes

import (
	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/ledger/globalpath"
)

func runJumpShortOnInputContext(e *engine.Engine, d []byte) {
	mustParLen(d, 1)
	if globalpath.IsConsumedOutputContext(e.RegValue(engine.RegInvocationPath)) {
		e.Move(int(d[0]))
	} else {
		e.Move(1)
	}
}

func runJumpLongOnInputContext(e *engine.Engine, d []byte) {
	mustParLen(d, 2)
	if globalpath.IsConsumedOutputContext(e.RegValue(engine.RegInvocationPath)) {
		e.Move(int(easyutxo.DecodeInteger[uint16](d[:2])))
	} else {
		e.Move(1 + 1)
	}
}

func runJumpShortOnTrue(e *engine.Engine, d []byte) {
	runJump(e, d, false, false)
}

func runJumpLongOnTrue(e *engine.Engine, d []byte) {
	runJump(e, d, false, true)
}

func runJumpShortOnFalse(e *engine.Engine, d []byte) {
	runJump(e, d, true, false)
}

func runJumpLongOnFalse(e *engine.Engine, d []byte) {
	runJump(e, d, true, true)
}

func runJump(e *engine.Engine, d []byte, onFalse, isLong bool) {
	isFalse := e.IsFalse()
	var offs int
	switch {
	case onFalse == isFalse && isLong:
		mustParLen(d, 2)
		offs = int(easyutxo.DecodeInteger[uint16](d[:2]))
	case onFalse == isFalse && !isLong:
		mustParLen(d, 1)
		offs = int(d[0])
	case onFalse != isFalse && isLong:
		mustParLen(d, 2)
		offs = 1 + 1
	case onFalse != isFalse && !isLong:
		mustParLen(d, 1)
		offs = 1
	}
	e.Move(offs)
	e.Pop()
}
