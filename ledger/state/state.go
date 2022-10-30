package state

import (
	"sync"
	"time"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraint"
	"github.com/lunfardo314/easyutxo/ledger/indexer"
)

// FinalState is a ledger state
type FinalState struct {
	mutex *sync.RWMutex
	store ledger.StateStore
}

func NewLedgerState(store ledger.StateStore, genesisAddr constraint.AddressED25519, initialSupply uint64) *FinalState {
	outBytes, oid := genesisOutput(genesisAddr, initialSupply, uint32(time.Now().Unix()))
	batch := store.BatchedWriter()
	batch.Set(oid[:], outBytes)
	if err := batch.Commit(); err != nil {
		panic(err)
	}
	return &FinalState{
		mutex: &sync.RWMutex{},
		store: store,
	}
}

// NewInMemory mostly for testing
func NewInMemory(genesisAddr constraint.AddressED25519, initialSupply uint64) *FinalState {
	return NewLedgerState(common.NewInMemoryKVStore(), genesisAddr, initialSupply)
}

func genesisOutput(genesisAddr constraint.AddressED25519, initialSupply uint64, ts uint32) ([]byte, ledger.OutputID) {
	easyfl.Assert(initialSupply > 0, "initialSupply > 0")
	amount := constraint.NewAmount(initialSupply)
	timestamp := constraint.NewTimestamp(ts)
	ret := lazyslice.EmptyArray()
	ret.Push(amount.Bytes())
	ret.Push(timestamp.Bytes())
	ret.Push(genesisAddr.Bytes())
	return ret.Bytes(), ledger.OutputID{}
}

func (u *FinalState) AddTransaction(txBytes []byte, traceOption ...int) ([]*indexer.IndexEntry, error) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	ctx, err := ValidationContextFromTransaction(txBytes, u, traceOption...)
	if err != nil {
		return nil, err
	}
	indexerUpdate, err := ctx.Validate()
	if err != nil {
		return nil, err
	}
	return indexerUpdate, u.updateLedger(ctx)
}

func (u *FinalState) GetUTXO(oid *ledger.OutputID) ([]byte, bool) {
	ret := u.store.Get(oid.Bytes())
	if len(ret) == 0 {
		return nil, false
	}
	return ret, true
}

func (u *FinalState) updateLedger(ctx *ValidationContext) error {
	batch := u.store.BatchedWriter()
	var err error
	// delete consumed outputs from the ledger and from accounts
	ctx.ForEachInputID(func(idx byte, oid *ledger.OutputID) bool {
		batch.Set(oid[:], nil)
		return true
	})
	if err != nil {
		return err
	}
	// add new outputs to the ledger and to accounts
	txID := ctx.TransactionID()
	ctx.tree.ForEach(func(idx byte, outputData []byte) bool {
		oid := ledger.NewOutputID(txID, idx)
		batch.Set(oid[:], outputData)
		return true
	}, Path(TransactionBranch, TxOutputBranch))
	return batch.Commit()
}
