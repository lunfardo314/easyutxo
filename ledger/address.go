package ledger

import (
	"crypto/ed25519"
	"encoding/hex"

	"github.com/lunfardo314/easyutxo/easyfl"
)

type Address []byte

func AddressFromED25519PubKey(pubKey ed25519.PublicKey) Address {
	return easyfl.MustEvalFromSource(nil, "addrED25519FromPubKey($0)", pubKey)
}

func (a Address) String() string {
	return hex.EncodeToString(a)
}
