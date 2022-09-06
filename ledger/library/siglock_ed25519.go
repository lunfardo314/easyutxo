package library

import (
	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/ledger/opcodes"
)

var SigLockED25519 = opcodes.MustGenProgram(func(p *engine.Program) {
	p.OP(opcodes.OpsPushFromReg).B(engine.RegInvocationData)         // load address into the stack
	p.OP(opcodes.OpsJumpShortOnInputContext).TargetShort("checksig") // Jump to 'checksig' if input context (signature checking)
	// --- Continues on transaction context
	p.OP(opcodes.OpsEqualLenShort).B(32) // Checks if the length of invocation data is equal to 32
	p.OP(opcodes.OpsExit)                // ends script here. Fails if length is wrong
	p.Label("checksig")
	// ---- here we have invocation context inputs
	p.OP(opcodes.OpsMakeUnlockBlockPathToReg).B(engine.FirstWriteableRegister)     // make path of the corresponding unlock-block into the register
	p.OP(opcodes.OpsPushBytesFromPathAndIndex).B(engine.FirstWriteableRegister, 0) // push #0 element of the unlock-block
	p.OP(opcodes.OpsEqualLenShort).B(0)                                            // Checks if the first element is zero length
	p.OP(opcodes.OpsJumpLongOnFalse).TargetShort("refinput")                       // jumps to 'refinput' is first element not 0
	// ---- here we are checking the signature
	p.OP(opcodes.OpsPushBytesFromPathAndIndex).B(engine.FirstWriteableRegister, 2) // push #2 element of the unlock-block with public key
	p.OP(opcodes.OpsPushBytesFromPathAndIndex).B(engine.FirstWriteableRegister, 1) // push #1 element of the unlock-block with signature
	p.OP(opcodes.OpsPushTransactionEssenceBytes)                                   // push essence bytes
	p.OP(opcodes.OpsSigLockED25519)                                                // check signature
	p.OP(opcodes.OpsJumpLongOnTrue).TargetShort("sigok")                           // jumps to 'sigok' if signature correct
	p.OP(opcodes.OpsExit)                                                          // ends script here. Fails if signature is invalid
	p.Label("sigok")
	// --- here signature is valid
	p.OP(opcodes.OpsPop)           // remove essence bytes. Signature and public key is left
	p.OP(opcodes.OpsPop)           // remove signature. Public key is left
	p.OP(opcodes.OpsBlake2b)       // has the public key. Replace with hash. Now 2 top elements of the stack are hash and address
	p.OP(opcodes.OpsEqualStackTop) // compares public key hash with address
	p.OP(opcodes.OpsExit)          // ends script here. Fails if public key has not equal to address
	p.Label("refinput")
	// ---- unlock block contains reference to another
	// TODO
})

// Each input ID has an input under ths same long index
// Each input has unlock block under the same long index
// Each unlock block is LazyTree, interpreted up to scripts
