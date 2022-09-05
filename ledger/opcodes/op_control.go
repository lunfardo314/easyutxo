package opcodes

import "github.com/lunfardo314/easyutxo/engine"

func jump8OnTrueRunner(e *engine.Engine, d []byte) bool {
	mustParLen(d, 1)
	if e.IsFalse() {
		return true
	}
	e.Pop()
	return false
}
