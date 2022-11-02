package library

import (
	"crypto"
	"crypto/ed25519"
	"math/rand"
	"time"

	"github.com/lunfardo314/easyfl"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func UnlockParamsBySignatureED25519(msg []byte, privKey ed25519.PrivateKey) []byte {
	sig, err := privKey.Sign(rnd, msg, crypto.Hash(0))
	easyfl.AssertNoError(err)
	pubKey := privKey.Public().(ed25519.PublicKey)
	return easyfl.Concat(sig, []byte(pubKey))
}
