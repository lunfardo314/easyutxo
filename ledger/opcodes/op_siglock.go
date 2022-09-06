package opcodes

import "github.com/lunfardo314/easyutxo/engine"

func runSigLogED25519(e *engine.Engine, d []byte) {
	mustParLen(d, 0)
	e.PushBool(false)
	e.Move(1)
}
