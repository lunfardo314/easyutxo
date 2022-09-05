package opcodes

import "github.com/lunfardo314/easyutxo/engine"

func opSigED25519Runner(e *engine.Engine, d []byte) {
	mustParLen(d, 0)
	e.PushBool(false)
	e.Move(1)
}
