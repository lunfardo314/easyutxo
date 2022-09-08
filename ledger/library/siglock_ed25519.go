package library

import (
	"github.com/lunfardo314/easyutxo/ledger/opcodes"
)

var SigLockED25519 = opcodes.MustCompileSource(SigLockED25519Source)

//	opcodes.MustGenProgram(func(p *engine.Program) {
//	p.Opcode(opcodes.OpsPushFromReg).ParamBytes(engine.RegInvocationData) // load address into the stack
//	p.Opcode(opcodes.OpsJumpShortOnInputContext).TargetShort("checksig")  // Jump to 'checksig' if input context (signature checking)
//	// --- Continues on transaction context
//	p.Opcode(opcodes.OpsEqualLenShort).ParamBytes(32) // Checks if the length of invocation data is equal to 32
//	p.Opcode(opcodes.OpsExit)                         // ends script here. Fails if length is wrong
//	p.Label("checksig")
//	// ---- here we have invocation context inputs
//	p.Opcode(opcodes.OpsMakeUnlockBlockPathToReg).ParamBytes(engine.FirstWriteableRegister)     // make globalpath of the corresponding unlock-block into the register
//	p.Opcode(opcodes.OpsPushBytesFromPathAndIndex).ParamBytes(engine.FirstWriteableRegister, 0) // push #0 element of the unlock-block
//	p.Opcode(opcodes.OpsEqualLenShort).ParamBytes(0)                                            // Checks if the first element is zero length
//	p.Opcode(opcodes.OpsJumpShortOnFalse).TargetShort("refinput")                               // jumps to 'refinput' is first element not 0
//	// ---- here we are checking the signature
//	p.Opcode(opcodes.OpsPushBytesFromPathAndIndex).ParamBytes(engine.FirstWriteableRegister, 2) // push #2 element of the unlock-block with public key
//	p.Opcode(opcodes.OpsPushBytesFromPathAndIndex).ParamBytes(engine.FirstWriteableRegister, 1) // push #1 element of the unlock-block with signature
//	p.Opcode(opcodes.OpsPushTransactionEssenceBytes)                                            // push essence bytes
//	p.Opcode(opcodes.OpsVerifySigED25519)                                                       // check signature
//	p.Opcode(opcodes.OpsJumpShortOnTrue).TargetShort("sigok")                                   // jumps to 'sigok' if signature correct
//	p.Opcode(opcodes.OpsExit)                                                                   // ends script here. Fails if signature is invalid
//	p.Label("sigok")
//	// --- here signature is valid
//	p.Opcode(opcodes.OpsPop)           // remove essence bytes. Signature and public key is left
//	p.Opcode(opcodes.OpsPop)           // remove signature. Public key is left
//	p.Opcode(opcodes.OpsBlake2b)       // has the public key. Replace with hash. Now 2 top elements of the stack are hash and address
//	p.Opcode(opcodes.OpsEqualStackTop) // compares public key hash with address
//	p.Opcode(opcodes.OpsExit)          // ends script here. Fails if public key has not equal to address
//	p.Label("refinput")
//	// ---- unlock block contains reference to another
//	// TODO
//})

// Each input ID has an input under ths same long index
// Each input has unlock-block under the same long index
// Each unlock block is LazyTree, interpreted up to scripts

var SigLockED25519Source = `
	reg->stack 1 				; push address from register #1 into the stack
	reg->stack 0                ; push invocation path
	[:]==param 0,1,u8/1         ; checks if the 0 byte of the invocation path is equal to 1 (consumed context) 
	ifInputContext> checksig    ; Jump to 'checksig' if it is consumed context (signature checking)
	; -------------------------- here just check if invocation data is 32 byte-long 
	size16->stack				; push size to stack as uint16
	[:]==param 0,2,u16.32       ; compare size with 32 
	exit						; ends script here. Fails if false is at the top, i.e. length is not 32
	> checksig					
	; -------------------------- here we have input invocation context (stack == (invocation path, address)
	[:]->stack 2,4              ; last 2 bytes of the invocation path to the stack
	param->stack hex/0000		; prefix of the unlock blocks
	concatReplace 2				; make corresponding unlock block path
	reg->stack 2				; invocation index from reg 2 to stack
	pushFromPathIndex           ; take element at path(top-1) and index(top) and replace index with it.
	size16->stack				; Unlock-block now is in the stack. Push size on top
	[:]==param 0,1,u8/3         ; checks if size of unlock block is 3. It means it is reference block
	ifTrue> refinput			; go to check referenced input if seize of unlock-block is 3
	; -------------------------- here we expect signature and public key
	[:]->stack 64,96			; put next 32 bytes into stack (public key)
	swap                        ; make unlock-block on top
	[:]->stack 0,64				; put first 64 bytes into stack (signature)
	swap						; make unlock-block on top
	drop 1                      ; drop unlock block
	pushFromPathParam hex/0001  ; bytes of input IDs
	pushFromPathParam hex/0002  ; bytes of outputs
	pushFromPathParam hex/0003  ; bytes of timestamp
	pushFromPathParam hex/0004  ; bytes of input commitment
	concatReplace 4				; make transaction essence bytes to stack	
	verifySigED25519            ; verify the signature of essence against public key
	ifTrue> sigok				; check if signature was ok
	exit                        ; signature not ok, leave with false (fail)
	> sigok
	; --------------------------- here signature is ok, now checking if it is the right public key
	drop 2                      ; remove essence and signature bytes. Signature and public key is left
	blake2b                     ; hash the public key, replace. Now 2 top elements of the stack are hash and address
    ==                          ; compares public key hash with address
	exit                        ; ends script here. Fails if public key has not equal to address
	> refinput  				
	; --------------------------- here we are checking referenced input if it unlocks the current one
	drop 1                      ; drop the size from the top/ Now unlock-block is on the top  
	loadRefInputBlock           ; load the referenced block of consumed input
	
`
