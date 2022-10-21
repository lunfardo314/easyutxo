package ledger

import (
	"crypto"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"golang.org/x/crypto/blake2b"
)

type (
	AddressType byte
	Address     []byte
)

const AddressED25519 = AddressType(ConstraintSigLockED25519)

func (at AddressType) DataSize() int {
	switch at {
	case AddressED25519:
		return 32
	}
	panic("unknown address type")
}

func (at AddressType) String() string {
	switch at {
	case AddressED25519:
		return "AddressED25519"
	}
	return "unknown address type"
}

func AddressFromBytes(data []byte) (Address, error) {
	if len(data) == 0 {
		return nil, io.EOF
	}
	if len(data) != AddressType(data[0]).DataSize()+1 {
		return nil, fmt.Errorf("AddressFromBytes: wrong data size")
	}
	return data, nil
}

func AddressFromED25519PubKey(pubKey ed25519.PublicKey) Address {
	d := blake2b.Sum256(pubKey)
	return common.Concat(byte(AddressED25519), d[:])
}

func AddressED25519Null() Address {
	var empty [32]byte
	return common.Concat(byte(AddressED25519), empty[:])
}

func (a Address) Type() AddressType {
	return AddressType(a[0])
}

func (a Address) Bytes() []byte {
	return a
}

func (a Address) String() string {
	return fmt.Sprintf("%s(%s)", a.Type(), hex.EncodeToString(a[1:]))
}

func UnlockDataByReference(ref byte) []byte {
	return []byte{ref}
}

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func UnlockDataBySignatureED25519(msg []byte, privKey ed25519.PrivateKey) []byte {
	sig, err := privKey.Sign(rnd, msg, crypto.Hash(0))
	easyfl.AssertNoError(err)
	pubKey := privKey.Public().(ed25519.PublicKey)
	return easyfl.Concat(sig, []byte(pubKey))
}
