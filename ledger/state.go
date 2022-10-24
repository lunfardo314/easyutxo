package ledger

import (
	"crypto/ed25519"
	"fmt"
	"sync"
	"time"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

type StateAccess interface {
	GetUTXO(id *OutputID) ([]byte, bool)
	// GetUTXOsForAddress order non-deterministic
	GetUTXOsForAddress(addr Lock) ([]*OutputWithID, error)
}

type KVStore interface {
	common.KVReader
	common.BatchedUpdatable
	common.Traversable
}

// State is a ledger state
type State struct {
	mutex *sync.RWMutex
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
	batch.Set(common.Concat(PartitionAccounts, out.Lock, oid[:]), []byte{0xff})
	if err := batch.Commit(); err != nil {
		panic(err)
	}
	return &State{
		mutex: &sync.RWMutex{},
		store: store,
	}
}

// NewLedgerStateInMemory mostly for testing
func NewLedgerStateInMemory(genesisPublicKey ed25519.PublicKey, initialSupply uint64) *State {
	return NewLedgerState(common.NewInMemoryKVStore(), genesisPublicKey, initialSupply)
}

func genesisOutput(genesisPublicKey ed25519.PublicKey, initialSupply uint64, ts uint32) (*Output, OutputID) {
	easyfl.Assert(initialSupply > 0, "initialSupply > 0")
	genesisLock := LockFromED25519PubKey(genesisPublicKey)
	out := NewOutput(initialSupply, ts, genesisLock)
	// genesis OutputID is all-0
	return out, OutputID{}
}

func (u *State) AddTransaction(txBytes []byte, traceOption ...int) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	ctx, err := ValidationContextFromTransaction(txBytes, u, traceOption...)
	if err != nil {
		return err
	}
	if err = ctx.Validate(); err != nil {
		return err
	}
	return u.updateLedger(ctx)
}

func (u *State) GetUTXO(id *OutputID) ([]byte, bool) {
	ret := u.store.Get(common.Concat(PartitionUTXO, id.Bytes()))
	if len(ret) == 0 {
		return nil, false
	}
	return ret, true
}

func (u *State) GetUTXOsForAddress(addr Lock) ([]*OutputWithID, error) {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	ret := make([]*OutputWithID, 0)
	prefix := common.Concat(PartitionAccounts, addr)
	var err error
	var oid OutputID
	var out *Output
	u.store.Iterator(prefix).Iterate(func(k, v []byte) bool {
		oid, err = OutputIDFromBytes(k[len(prefix):])
		if err != nil {
			return false
		}
		outBin, found := u.GetUTXO(&oid)
		if !found {
			err = fmt.Errorf("can't find output %s in address %s", oid, addr)
			return false
		}
		out, err = OutputFromBytes(outBin)
		if err != nil {
			return false
		}
		ret = append(ret, &OutputWithID{
			ID:     oid,
			Output: out,
		})
		return true
	})
	return ret, err
}

func (u *State) updateLedger(ctx *ValidationContext) error {
	batch := u.store.BatchedWriter()
	var err error
	// delete consumed outputs from the ledger and from accounts
	ctx.ForEachInputID(func(idx byte, o *OutputID) bool {
		consumedKey := common.Concat(PartitionUTXO, o[:])
		consumed := ctx.ConsumedOutput(idx)

		if len(u.store.Get(consumedKey)) == 0 {
			// only can happen if two threads are reading/writing to the same account. Not a normal situation
			err = fmt.Errorf("reace condition while deleting output from address %s", consumed.Lock)
			return false
		}
		batch.Set(consumedKey, nil)
		batch.Set(common.Concat(PartitionAccounts, consumed.Lock, o[:]), nil)
		return true
	})
	if err != nil {
		return err
	}
	// add new outputs to the ledger and to accounts
	txID := ctx.TransactionID()
	ctx.ForEachOutput(Path(TransactionBranch, TxOutputBranch), func(o *Output, _ uint32, outputPath lazyslice.TreePath) bool {
		id := NewOutputID(txID, outputPath[2])
		batch.Set(common.Concat(PartitionUTXO, id[:]), o.Bytes())
		batch.Set(common.Concat(PartitionAccounts, o.Lock, id[:]), []byte{0xff})
		return true
	})
	return batch.Commit()
}
