package opcodes

import "github.com/lunfardo314/easyutxo/engine"

func opSigED25519Runner(_ *engine.Engine, _ []byte) bool {
	return false
}
