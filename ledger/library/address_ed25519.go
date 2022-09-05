package library

import (
	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/ledger/opcodes"
)

var AddressED25519LockScript = opcodes.MustGenProgram(func(p *engine.Program) {
	// invocation data is 32 byte hash of the public key
	// When run in output context, it just checks the length of invocation data
})
