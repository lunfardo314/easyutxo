package ledger

import (
	"bytes"
	"crypto/ed25519"
	"math/rand"
	"time"

	"github.com/iotaledger/trie.go/common"
)

type UTXODB struct {
	store common.KVStore
}

func NewUTXODB(store common.KVStore, genesisPublicKey ed25519.PublicKey, initialSupply uint64) *UTXODB {
	out, oid := genesisOutput(genesisPublicKey, initialSupply, uint32(time.Now().Unix()))
	ret := &UTXODB{
		store: store,
	}
	ret.utxoPartition().Set(oid[:], out.Bytes())
	return ret
}

func NewUTXODBInMemory(genesisPublicKey ed25519.PublicKey, initialSupply uint64) *UTXODB {
	return NewUTXODB(common.NewInMemoryKVStore(), genesisPublicKey, initialSupply)
}

const supplyForTesting = uint64(1_000_000_000_000)

func NewUTXODBForTesting(store common.KVStore) (*UTXODB, ed25519.PrivateKey, ed25519.PublicKey) {
	privKey, pubKey := newKeyPair()
	ret := NewUTXODBInMemory(pubKey, supplyForTesting)
	return ret, privKey, pubKey
}

func genesisOutput(genesisPublicKey ed25519.PublicKey, initialSupply uint64, ts uint32) (*Output, OutputID) {
	addrData := AddressDataFromED25519PubKey(genesisPublicKey)
	common.Assert(initialSupply > 0, "initialSupply > 0")
	out := NewOutput()
	out.PutMainConstraint(ts, initialSupply)
	out.PutAddress(addrData, ConstraintSigLockED25519)
	return out, OutputID{}
}

func (u *UTXODB) utxoPartition() common.KVStore {
	return u.store
}

func (u *UTXODB) AddTransaction(txBytes []byte) error {
	ctx, err := TransactionContextFromTransaction(txBytes, u)
	if err != nil {
		return err
	}
	if err = ctx.Validate(); err != nil {
		return err
	}
	u.updateLedger(ctx.Transaction())
	return nil
}

func (u *UTXODB) GetUTXO(id *OutputID) (OutputData, bool) {
	ret := u.utxoPartition().Get(id.Bytes())
	if len(ret) == 0 {
		return nil, false
	}
	return ret, true
}

func (u *UTXODB) GetUTXOsForAddress(addr []byte) []OutputData {
	ret := make([]OutputData, 0)
	u.utxoPartition().Iterate(func(k, v []byte) bool {
		if bytes.Equal(addr, OutputFromBytes(v).Address()) {
			ret = append(ret, v)
		}
		return true
	})
	return ret
}

// updateLedger in the future must be atomic
func (u *UTXODB) updateLedger(tx *Transaction) {
	tx.ForEachInputID(func(_ byte, o OutputID) bool {
		u.utxoPartition().Set(o[:], nil)
		return true
	})
	// add new outputs
	txid := tx.ID()
	tx.ForEachOutput(func(o *Output, idx byte) bool {
		id := NewOutputID(txid, idx)
		u.utxoPartition().Set(id[:], o.Bytes())
		return true
	})
}

var rnd = rand.NewSource(time.Now().UnixNano())

func newKeyPair() (ed25519.PrivateKey, ed25519.PublicKey) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.New(rnd))
	if err != nil {
		panic(err)
	}
	return privKey, pubKey
}
