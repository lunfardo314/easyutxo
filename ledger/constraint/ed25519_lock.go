package constraint

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"golang.org/x/crypto/blake2b"
)

func AddressED25519SigLock(pubKey ed25519.PublicKey) []byte {
	d := blake2b.Sum256(pubKey)
	src := fmt.Sprintf("addressED25519(0x%s)", hex.EncodeToString(d[:]))
	_, _, binCode, err := easyfl.CompileExpression(src)
	easyfl.AssertNoError(err)
	return binCode
}

func AddressED25519SigLockNull() []byte {
	var empty [32]byte
	src := fmt.Sprintf("addressED25519(0x%s)", hex.EncodeToString(empty[:]))
	_, _, binCode, err := easyfl.CompileExpression(src)
	easyfl.AssertNoError(err)
	return binCode
}

func initAddressED25519Constraint() {
	example := AddressED25519SigLockNull()
	prefix, args, err := easyfl.ParseCallWithConstants(example, 1)
	easyfl.AssertNoError(err)
	common.Assert(len(args[0]) == 32, "len(args[0]) == 32")
	registerConstraint("addressED25519", prefix)
}

func ParseAddressED25519Constraint(data []byte) ([]byte, bool) {
	prefix, args, err := easyfl.ParseCallWithConstants(data, 1)
	if err != nil {
		return nil, false
	}
	prefix1, ok := PrefixByName("addressED25519")
	common.Assert(ok, "no addressED25519")
	if !bytes.Equal(prefix, prefix1) {
		return nil, false
	}
	if len(args[0]) != 32 {
		return nil, false
	}
	return args[0], true
}

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func UnlockParamsBySignatureED25519(msg []byte, privKey ed25519.PrivateKey) []byte {
	sig, err := privKey.Sign(rnd, msg, crypto.Hash(0))
	easyfl.AssertNoError(err)
	pubKey := privKey.Public().(ed25519.PublicKey)
	return easyfl.Concat(sig, []byte(pubKey))
}

const AddressED25519ConstraintSource = `

// ED25519 address constraint is 1-byte type and 32 bytes address, blake2b hash of the public key
// ED25519 unlock parameters expected to be 96 bytes-long

// takes ED25519 signature from unlock parameters, first 64 bytes 
func signatureED25519: slice($0, 0, 63) 

// takes ED25519 public key from unlock parameters
func publicKeyED25519: slice($0, 64, 95) // the rest 32 bytes is public key 

// 'unlockedWithSigED25519' specifies unlock constraint with the unlock params signature
// the signature must be valid and hash of the public key must be equal to the provided address

// $0 = constraint data (address data)
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

// $0 - ED25519 address, 32 byte blake2b hash of the public key
func addressED25519: or(
	and(
		isProducedBranch(@), 
		equal(len8($0), 32) 
	),
    and(
		isConsumedBranch(@), 
		or(                                    
				// if it is unlocked with reference, the signature is not checked
			unlockedByReference(selfConstraint, selfUnlockParameters, selfOutputIndex, selfReferencedConstraint),
				// otherwise signature is checked
			unlockedWithSigED25519($0, selfUnlockParameters)    
		)
	)
)
`
