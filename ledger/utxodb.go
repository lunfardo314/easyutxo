package ledger

import (
	"crypto/ed25519"
	"math/rand"
	"time"

	"github.com/iotaledger/trie.go/common"
)

type KVStore interface {
	common.KVReader
	common.BatchedUpdatable
	common.Traversable
}

type UTXODB struct {
	store KVStore
}

const (
	PartitionUTXO = byte(iota)
	PartitionTrie
	PartitionAccounts
)

func NewUTXODB(store KVStore, genesisPublicKey ed25519.PublicKey, initialSupply uint64) *UTXODB {
	out, oid := genesisOutput(genesisPublicKey, initialSupply, uint32(time.Now().Unix()))
	batch := store.BatchedWriter()
	batch.Set(common.Concat(PartitionUTXO, oid[:]), out.Bytes())
	batch.Set(common.Concat(PartitionAccounts, out.Address(), oid[:]), []byte{0xff})
	if err := batch.Commit(); err != nil {
		panic(err)
	}
	return &UTXODB{store}
}

func NewUTXODBInMemory(genesisPublicKey ed25519.PublicKey, initialSupply uint64) *UTXODB {
	return NewUTXODB(common.NewInMemoryKVStore(), genesisPublicKey, initialSupply)
}

const supplyForTesting = uint64(1_000_000_000_000)

func NewUTXODBForTesting() (*UTXODB, ed25519.PrivateKey, ed25519.PublicKey) {
	originPrivKey, originPubKey := newKeyPair()
	ret := NewUTXODBInMemory(originPubKey, supplyForTesting)
	return ret, originPrivKey, originPubKey
}

func genesisOutput(genesisPublicKey ed25519.PublicKey, initialSupply uint64, ts uint32) (*Output, OutputID) {
	addrData := AddressDataFromED25519PubKey(genesisPublicKey)
	common.Assert(initialSupply > 0, "initialSupply > 0")
	out := NewOutput()
	out.PutMainConstraint(ts, initialSupply)
	out.PutAddress(addrData, ConstraintSigLockED25519)
	// genesis OutputID is all-0
	return out, OutputID{}
}

func (u *UTXODB) AddTransaction(txBytes []byte) error {
	ctx, err := TransactionContextFromTransaction(txBytes, u)
	if err != nil {
		return err
	}
	if err = ctx.Validate(); err != nil {
		return err
	}
	u.updateLedger(ctx)
	return nil
}

func (u *UTXODB) GetUTXO(id *OutputID) (OutputData, bool) {
	ret := u.store.Get(common.Concat(PartitionUTXO, id.Bytes()))
	if len(ret) == 0 {
		return nil, false
	}
	return ret, true
}

func (u *UTXODB) GetUTXOsForAddress(addr []byte) []OutputData {
	ret := make([]OutputData, 0)
	iterator := u.store.Iterator(common.Concat(PartitionAccounts, addr))
	iterator.Iterate(func(k, v []byte) bool {
		ret = append(ret, v)
		return true
	})
	return ret
}

func (u *UTXODB) updateLedger(ctx *TransactionContext) {
	batch := u.store.BatchedWriter()
	// delete consumed outputs from the ledger and from accounts
	tx := ctx.Transaction()
	tx.ForEachInputID(func(idx byte, o OutputID) bool {
		batch.Set(common.Concat(PartitionUTXO, o[:]), nil)
		consumed := ctx.ConsumedOutput(idx)
		addr := consumed.Address()
		batch.Set(common.Concat(PartitionAccounts, addr, o[:]), nil)
		return true
	})
	// add new outputs to the ledger and to accounts
	txID := tx.ID()
	tx.ForEachOutput(func(o *Output, idx byte) bool {
		id := NewOutputID(txID, idx)
		batch.Set(common.Concat(PartitionUTXO, id[:]), o.Bytes())
		batch.Set(common.Concat(PartitionAccounts, o.Address(), id[:]), []byte{0xff})
		return true
	})
	if err := batch.Commit(); err != nil {
		panic(err)
	}
}

var rnd = rand.NewSource(time.Now().UnixNano())

func newKeyPair() (ed25519.PrivateKey, ed25519.PublicKey) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.New(rnd))
	if err != nil {
		panic(err)
	}
	return privKey, pubKey
}
