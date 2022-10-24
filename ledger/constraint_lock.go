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

const LockAddressED25519 = LockType(ConstraintTypeSigLockED25519)

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

// ED25519 unlock parameters expected to be 96 bytes-long

// takes ED25519 signature from unlock parameters 
func selfSignatureED25519: slice(selfUnlockParameters, 0, 63) 

// takes ED25519 public key from unlock parameters
func selfPublicKeyED25519: slice(selfUnlockParameters, 64, 95)

// 'selfUnlockedWithSigED25519' specifies unlock constraint with the unlock block signature
// the signature must be valid and hash of the public key must be equal to the provided address
func selfUnlockedWithSigED25519: and(
	equal(len8(selfUnlockParameters), 96),      // unlock block must be 96 bytes long
	validSignatureED25519(
		txEssenceBytes,                    // function 'txEssenceHash' returns transaction essence bytes 
		selfSignatureED25519,              // first 64 bytes is signature
		selfPublicKeyED25519               // the rest is public key
	),
	equal(
		selfConstraintData,                    // address in the constraint data must be equal to the hash of the  
		addrDataED25519FromPubKey(             // public key
			selfPublicKeyED25519
		)
	)
)

// 'selfUnlockedWithReference'' specifies validation of the input unlock with the reference
func selfUnlockedWithReference: and(
	equal(len8(selfUnlockParameters), 1),                // unlock block must be 2 bytes long
	lessThan(selfUnlockParameters, selfOutputIndex),     // unlock block must point to another input with 
														 // strictly smaller index. This prevents reference cycles	
	equal(selfConstraint, selfReferencedConstraint)      // the referenced constraint bytes must be equal to the
														 // self constrain bytes
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
			selfUnlockedWithReference,    // if it is unlocked with reference, the signature is not checked
			selfUnlockedWithSigED25519    // otherwise signature is checked
		)
	)
)
`
