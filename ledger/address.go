package ledger

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/iotaledger/trie.go/common"
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

func (a Address) Type() AddressType {
	return AddressType(a[0])
}

func (a Address) Bytes() []byte {
	return a
}

func (a Address) String() string {
	return fmt.Sprintf("%s(%s)", a.Type(), hex.EncodeToString(a[1:]))
}
