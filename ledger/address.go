package ledger

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
)

type (
	AddressData []byte
	AddressType byte
	Address     struct {
		data       AddressData
		constraint AddressType
	}
)

func (at AddressType) String() string {
	switch byte(at) {
	case ConstraintSigLockED25519:
		return "AddressED25519"
	}
	return "unknown address type"
}

func AddressFromED25519PubKey(pubKey ed25519.PublicKey) *Address {
	return &Address{
		data:       easyfl.MustEvalFromSource(nil, "addrDataED25519FromPubKey($0)", pubKey),
		constraint: AddressType(ConstraintSigLockED25519),
	}
}

func (a Address) Bytes() []byte {
	return common.Concat(byte(a.constraint), []byte(a.data))
}

func (a Address) String() string {
	return fmt.Sprintf("%s(%s)", a.constraint, hex.EncodeToString(a.data))
}
