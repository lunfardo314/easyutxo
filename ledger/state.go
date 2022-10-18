package ledger

import (
	"crypto/ed25519"
	"time"

	"github.com/iotaledger/trie.go/common"
)

type StateAccess interface {
	GetUTXO(id *OutputID) (OutputData, bool)
	// GetUTXOsForAddress order non-deterministic
	GetUTXOsForAddress(addr Address) []OutputWithID
}

type KVStore interface {
	common.KVReader
	common.BatchedUpdatable
	common.Traversable
}

// State is a ledger state
type State struct {
	store KVStore
}

const (
	PartitionUTXO = byte(iota)
	PartitionAccounts
)

func NewLedgerState(store KVStore, genesisPublicKey ed25519.PublicKey, initialSupply uint64) *State {
	out, oid := genesisOutput(genesisPublicKey, initialSupply, uint32(time.Now().Unix()))
	batch := store.BatchedWriter()
	batch.Set(common.Concat(PartitionUTXO, oid[:]), out.Bytes())
	batch.Set(common.Concat(PartitionAccounts, out.Address(), oid[:]), []byte{0xff})
	if err := batch.Commit(); err != nil {
		panic(err)
	}
	return &State{store}
}

// NewLedgerStateInMemory mostly for testing
func NewLedgerStateInMemory(genesisPublicKey ed25519.PublicKey, initialSupply uint64) *State {
	return NewLedgerState(common.NewInMemoryKVStore(), genesisPublicKey, initialSupply)
}

func genesisOutput(genesisPublicKey ed25519.PublicKey, initialSupply uint64, ts uint32) (*Output, OutputID) {
	addr := AddressFromED25519PubKey(genesisPublicKey)
	common.Assert(initialSupply > 0, "initialSupply > 0")
	out := NewOutput()
	out.PutMainConstraint(ts, initialSupply)
	out.PutAddress(addr)
	// genesis OutputID is all-0
	return out, OutputID{}
}

func (u *State) AddTransaction(txBytes []byte) error {
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

func (u *State) GetUTXO(id *OutputID) (OutputData, bool) {
	ret := u.store.Get(common.Concat(PartitionUTXO, id.Bytes()))
	if len(ret) == 0 {
		return nil, false
	}
	return ret, true
}

func (u *State) GetUTXOsForAddress(addr Address) []OutputWithID {
	ret := make([]OutputWithID, 0)
	prefix := common.Concat(PartitionAccounts, addr)
	u.store.Iterator(prefix).Iterate(func(k, v []byte) bool {
		oid, err := OutputIDFromBytes(k[len(prefix):])
		common.AssertNoError(err)

		outBin, found := u.GetUTXO(&oid)
		common.Assert(found, "can't find output %s in address %s", oid, addr)

		ret = append(ret, OutputWithID{
			ID:     oid,
			Output: OutputFromBytes(outBin),
		})
		return true
	})
	return ret
}

func (u *State) updateLedger(ctx *TransactionContext) {
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
