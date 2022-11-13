package library

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/lunfardo314/easyfl"
	"golang.org/x/crypto/blake2b"
)

type AddressED25519 []byte

const (
	addressED25519Name     = "addressED25519"
	addressED25519Template = addressED25519Name + "(0x%s)"
)

func AddressED25519FromBytes(data []byte) (AddressED25519, error) {
	sym, _, args, err := easyfl.ParseBinaryOneLevel(data, 1)
	if err != nil {
		return nil, err
	}
	if sym != addressED25519Name {
		return nil, fmt.Errorf("not an AddressED25519")
	}
	addrBin := easyfl.StripDataPrefix(args[0])
	if len(addrBin) != 32 {
		return nil, fmt.Errorf("wrong data length")
	}
	return addrBin, nil
}

func AddressED25519FromPublicKey(pubKey ed25519.PublicKey) AddressED25519 {
	h := blake2b.Sum256(pubKey)
	return h[:]
}

func AddressED25519Null() AddressED25519 {
	return make([]byte, 32)
}

func (a AddressED25519) source() string {
	return fmt.Sprintf(addressED25519Template, hex.EncodeToString(a))
}

func (a AddressED25519) Bytes() []byte {
	return mustBinFromSource(a.source())
}

func (a AddressED25519) IndexableTags() []Accountable {
	return []Accountable{a}
}

func (a AddressED25519) UnlockableWith(acc AccountID, ts uint32) bool {
	return bytes.Equal(a.AccountID(), acc)
}

func (a AddressED25519) AccountID() AccountID {
	return a.Bytes()
}

func (a AddressED25519) Name() string {
	return addressED25519Name
}

func (a AddressED25519) String() string {
	return a.source()
}

func (a AddressED25519) AsLock() Lock {
	return a
}

func initAddressED25519Constraint() {
	easyfl.MustExtendMany(AddressED25519ConstraintSource)

	example := AddressED25519Null()
	addrBack, err := AddressED25519FromBytes(example.Bytes())
	easyfl.AssertNoError(err)
	easyfl.Assert(Equal(addrBack, AddressED25519Null()), "inconsistency "+addressED25519Name)

	prefix, err := easyfl.ParseCallPrefixFromBinary(example.Bytes())
	easyfl.AssertNoError(err)

	registerConstraint(addressED25519Name, prefix, func(data []byte) (Constraint, error) {
		return AddressED25519FromBytes(data)
	})
}

const AddressED25519ConstraintSource = `

// ED25519 address constraint wraps 32 bytes address, the blake2b hash of the public key

// $0 = address data 32 bytes
// $1 = signature
// $2 = public key
// return true if transaction essence signature is valid for the address 

func unlockedWithSigED25519: and(
	equal($0, blake2b($2)), 		       // address in the address data must be equal to the hash of the public key
	validSignatureED25519(txEssenceBytes, $1, $2)
)

// 'unlockedByReference'' specifies validation of the input unlock with the reference.
// The referenced constraint must be exactly the same  but with strictly lesser index.
// This prevents from cycles and forces some other unlock mechanism up in the list of outputs

func unlockedByReference: and(
	lessThan(selfUnlockParameters, selfOutputIndex),              // unlock parameter must point to another input with 
							                                      // strictly smaller index. This prevents reference cycles	
	equal(self, consumedLockByOutputIndex(selfUnlockParameters))  // the referenced constraint bytes must be equal to the self constraint bytes
)

// if it is 'produced' invocation context (constraint invoked in the input), only size of the address is checked
// Otherwise the first will check first condition if it is unlocked by reference, otherwise checks unlocking signature
// Second condition not evaluated if the first is true 

// $0 - ED25519 address, 32 byte blake2b hash of the public key
func addressED25519: and(
	equal(selfBlockIndex,2), // locks must be at block 2
	or(
		and(
			isPathToProducedOutput(@), 
			equal(len8($0), 32) 
		),
		and(
			isPathToConsumedOutput(@), 
			or(
					// if it is unlocked with reference, the signature is not checked
				unlockedByReference,
					// tx signature is checked
				unlockedWithSigED25519($0, signatureED25519(txSignature), publicKeyED25519(txSignature)) 
			)
		),
		!!!addressED25519_unlock_failed
	)
)

`
