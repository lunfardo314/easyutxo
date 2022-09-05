package library

import (
	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/ledger/opcodes"
)

var AddressED25519SigLock = opcodes.MustGenProgram(func(p *engine.Program) {
	// load address into the stack
	p.OP(opcodes.OPS_LOAD_FROM_REG).P(engine.RegInvocationData)
	// Jump if input context (signature checking)
	p.OP(opcodes.OPS_JUMP8_ON_INPUT_CTX).JS("checksig")
	// Continues on transaction context
	// Checks if the length of invocation data is equal to 32
	p.OP(opcodes.OPS_EQUAL_LEN8).P(32)
	// ends script here. Fails if length is wrong
	p.OP(opcodes.OPS_EXIT) // >>>>>>>>>>>>>>>>>>>>>>>
	p.L("checksig")
	p.OP(opcodes.OPS_SIGLOCK_ED25519)
})
