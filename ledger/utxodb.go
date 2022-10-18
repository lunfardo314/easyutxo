package ledger

import (
	"crypto/ed25519"
	"math/rand"
	"time"

	"github.com/iotaledger/trie.go/common"
)

// UTXODB is a ledger.State in-memory

const supplyForTesting = uint64(1_000_000_000_000)

func NewStateInMemory(genesisPublicKey ed25519.PublicKey, initialSupply uint64) *State {
	return NewState(common.NewInMemoryKVStore(), genesisPublicKey, initialSupply)
}

func NewUTXODBForTesting() (*State, ed25519.PrivateKey, ed25519.PublicKey) {
	originPrivKey, originPubKey := newKeyPair()
	ret := NewStateInMemory(originPubKey, supplyForTesting)
	return ret, originPrivKey, originPubKey
}

var rnd = rand.NewSource(time.Now().UnixNano())

func newKeyPair() (ed25519.PrivateKey, ed25519.PublicKey) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.New(rnd))
	if err != nil {
		panic(err)
	}
	return privKey, pubKey
}
