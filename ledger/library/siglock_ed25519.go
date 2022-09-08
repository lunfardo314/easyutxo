package library

import (
	"github.com/lunfardo314/easyutxo/ledger/opcodes"
)

var SigLockED25519 = opcodes.MustCompileSource(SigLockED25519Source)

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
	[:]->stack 0,1              ; first byte of the block
	reg->stack 3                ; load script invocation index
	== 							; compare the two
	ifTrue> refok1              ; must be equal
	exit                        ; fail if not
	> refok1
	; --------------------------- type of invication is the same, now we have ti check if address is the same
	drop 4                      ; remove last 4, address must be left
	loadRefInputBlock           ; load the referenced block of consumed input
	[:]->stack 1,33             ; load address
	==                          ; must be equal. Fauls if it is not
`
