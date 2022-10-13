package ledger

import (
	"crypto/ed25519"
	"encoding/hex"

	"github.com/lunfardo314/easyfl"
)

type AddressData []byte

func AddressDataFromED25519PubKey(pubKey ed25519.PublicKey) AddressData {
	return easyfl.MustEvalFromSource(nil, "addrED25519FromPubKey($0)", pubKey)
}

func (a AddressData) String() string {
	return hex.EncodeToString(a)
}
