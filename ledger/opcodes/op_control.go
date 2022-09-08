package opcodes

import (
	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/ledger/globalpath"
)

func runJumpShortOnInputContext(e *engine.Engine, p [][]byte) {
	if globalpath.IsConsumedOutputContext(e.RegValue(engine.RegInvocationPath)) {
		e.Move(int(p[0][0]))
	} else {
		e.Move(1 + 1)
	}
}

func runJumpLongOnInputContext(e *engine.Engine, p [][]byte) {
	if globalpath.IsConsumedOutputContext(e.RegValue(engine.RegInvocationPath)) {
		e.Move(int(easyutxo.DecodeInteger[uint16](p[0])))
	} else {
		e.Move(1 + 2)
	}
}

func runJumpShortOnTrue(e *engine.Engine, p [][]byte) {
	runJump(e, p, false, false)
}

func runJumpLongOnTrue(e *engine.Engine, p [][]byte) {
	runJump(e, p, false, true)
}

func runJumpShortOnFalse(e *engine.Engine, p [][]byte) {
	runJump(e, p, true, false)
}

func runJumpLongOnFalse(e *engine.Engine, p [][]byte) {
	runJump(e, p, true, true)
}

func runJump(e *engine.Engine, p [][]byte, onFalse, isLong bool) {
	isFalse := e.IsFalse()
	var offs int
	switch {
	case onFalse == isFalse && isLong:
		offs = int(easyutxo.DecodeInteger[uint16](p[0]))
	case onFalse == isFalse && !isLong:
		offs = int(p[0][0])
	case onFalse != isFalse && isLong:
		offs = 1 + 2
	case onFalse != isFalse && !isLong:
		offs = 1 + 1
	}
	e.Move(offs)
	e.Pop()
}
