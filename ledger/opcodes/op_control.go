package opcodes

import (
	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/engine"
)

func runJump8OnInputContext(e *engine.Engine, d []byte) {
	mustParLen(d, 1)
	if e.RegValue(engine.RegInvocationPath)[0] == 1 {
		// ledger.ValidationCtxInputsIndex
		e.Move(int(d[0]))
	} else {
		e.Move(1)
	}
}

func runJump16OnInputContext(e *engine.Engine, d []byte) {
	mustParLen(d, 2)
	if e.RegValue(engine.RegInvocationPath)[0] == 1 {
		// ledger.ValidationCtxInputsIndex
		e.Move(int(easyutxo.DecodeInteger[uint16](d[:2])))
	} else {
		e.Move(1 + 1)
	}
}

func runJump8OnTrue(e *engine.Engine, d []byte) {
	runJump(e, d, false, false)
}

func runJump16OnTrue(e *engine.Engine, d []byte) {
	runJump(e, d, false, true)
}

func runJump8OnFalse(e *engine.Engine, d []byte) {
	runJump(e, d, true, false)
}

func runJump16OnFalse(e *engine.Engine, d []byte) {
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
