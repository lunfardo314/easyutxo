package ledger

import (
	"crypto"
	"crypto/ed25519"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"golang.org/x/crypto/blake2b"
)

// Lock is normally an address with private key behind or an alias address
type (
	LockType byte
	Lock     []byte
)

// LockAddressED25519 ED25519 address is a special type of Lock
const (
	LockAddressED25519           = LockType(ConstraintTypeSigLockED25519)
	LockAddressED25519WithExpire = LockType(ConstraintTypeSigLockED25519WithExpire)
)

func (at LockType) DataSize() int {
	switch at {
	case LockAddressED25519:
		return 32
	}
	panic("unknown address type")
}

func (at LockType) String() string {
	switch at {
	case LockAddressED25519:
		return "LockAddressED25519"
	}
	return "unknown address type"
}

func LockFromBytes(data []byte) (Lock, error) {
	if len(data) == 0 {
		return nil, io.EOF
	}
	if len(data) != LockType(data[0]).DataSize()+1 {
		return nil, fmt.Errorf("LockFromBytes: wrong data size")
	}
	return data, nil
}

func LockFromED25519PubKey(pubKey ed25519.PublicKey) Lock {
	d := blake2b.Sum256(pubKey)
	return common.Concat(byte(LockAddressED25519), d[:])
}

func LockED25519Null() Lock {
	var empty [32]byte
	return common.Concat(byte(LockAddressED25519), empty[:])
}

func (a Lock) Type() LockType {
	return LockType(a[0])
}

func (a Lock) Bytes() []byte {
	return a
}

func (a Lock) String() string {
	return fmt.Sprintf("%s(%s)", a.Type(), easyfl.Fmt(a[1:]))
}

func UnlockParamsByReference(ref byte) []byte {
	return []byte{ref}
}

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func UnlockParamsBySignatureED25519(msg []byte, privKey ed25519.PrivateKey) []byte {
	sig, err := privKey.Sign(rnd, msg, crypto.Hash(0))
	easyfl.AssertNoError(err)
	pubKey := privKey.Public().(ed25519.PublicKey)
	return easyfl.Concat(sig, []byte(pubKey))
}

const SigLockED25519ConstraintSource = `

// ED25519 address constraint is 1-byte type and 32 bytes address, blake2b hash of the public key
// ED25519 unlock parameters expected to be 96 bytes-long

// takes ED25519 signature from unlock parameters, first 64 bytes 
func signatureED25519: slice($0, 0, 63) 

// takes ED25519 public key from unlock parameters
func publicKeyED25519: slice($0, 64, 95) // the rest 32 bytes is public key 

// 'unlockedWithSigED25519' specifies unlock constraint with the unlock params signature
// the signature must be valid and hash of the public key must be equal to the provided address

// $0 = constraint data (address without type)
// $1 = unlock parameters of 96 bytes long

func unlockedWithSigED25519: and(
	equal(len8($1), 96),         // unlock parameters must be 96 bytes long 
	validSignatureED25519(
		txEssenceBytes,          // function 'txEssenceHash' returns transaction essence bytes 
		signatureED25519($1),    
		publicKeyED25519($1)     
	),
	equal($0, blake2b(publicKeyED25519($1)) )  
			// address in the constraint data must be equal to the hash of the public key
)

// 'unlockedByReference'' specifies validation of the input unlock with the reference.
// The referenced constraint must be exactly the same  but with strictly lesser index.
// This prevents from cycles and forces some other unlock mechanism up in the list of outputs

// $0 - constraint (with type byte)
// $1 - unlock parameters 1-byte long
// $2 - index 1-byte of the selfOutput
// $3 - referenced constraint

func unlockedByReference: and(
	equal(len8($1), 1),         // unlock parameters must be 1 byte long
	lessThan($1, $2),           // unlock parameter must point to another input with 
							    // strictly smaller index. This prevents reference cycles	
	equal($0, $3)               // the referenced constraint bytes must be equal to the self constraint bytes
)

// if it is 'produced' invocation context (constraint invoked in the input), only size of the address is checked
// Otherwise the first will check first condition if it is unlocked by reference, otherwise checks unlocking signature
// Second condition not evaluated if the first is true 

func sigLockED25519: or(
	and(
		isProducedBranch(@), 
		equal( len8(selfConstraintData), 32) 
	),
    and(
		isConsumedBranch(@), 
		or(                                    
			unlockedByReference(selfConstraint, selfUnlockParameters, selfOutputIndex, selfReferencedConstraint),   // if it is unlocked with reference, the signature is not checked
			unlockedWithSigED25519(selfConstraintData, selfUnlockParameters)    // otherwise signature is checked
		)
	)
)
`
