package library

import (
	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/ledger/opcodes"
)

var AddressED25519SigLock = opcodes.MustGenProgram(func(p *engine.Program) {
	// load address into the stack
	p.OP(opcodes.OpsLoadFromReg).P(engine.RegInvocationData)
	// Jump if input context (signature checking)
	p.OP(opcodes.OpsJumpShortOnInputContext).JS("checksig")
	// Continues on transaction context
	// Checks if the length of invocation data is equal to 32
	p.OP(opcodes.OpsEqualLenShort).P(32)
	// ends script here. Fails if length is wrong
	p.OP(opcodes.OpsExit) // >>>>>>>>>>>>>>>>>>>>>>>
	p.L("checksig")
	p.OP(opcodes.OpsSigLockED25519)
})
