package indexer

import (
	"sync"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger"
)

type Indexer struct {
	mutex *sync.RWMutex
	store ledger.IndexerStore
}

type IndexEntry struct {
	AccountID []byte
	OutputID  ledger.OutputID
	Delete    bool
}

func NewIndexer(store ledger.IndexerStore) *Indexer {
	return &Indexer{
		mutex: &sync.RWMutex{},
		store: store,
	}
}

func (inr *Indexer) GetUTXOsForAddress(addr []byte, state ledger.StateAccess) ([]*ledger.OutputDataWithID, error) {
	inr.mutex.RLock()
	defer inr.mutex.RUnlock()

	ret := make([]*ledger.OutputDataWithID, 0)
	var err error
	var found bool
	inr.store.Iterator(addr).Iterate(func(k, v []byte) bool {
		o := &ledger.OutputDataWithID{}
		o.ID, err = ledger.OutputIDFromBytes(k[len(addr):])
		if err != nil {
			return false
		}
		o.OutputData, found = state.GetUTXO(&o.ID)
		if !found {
			// skip this output ID
			return true
		}
		ret = append(ret, o)
		return true
	})
	if err != nil {
		return nil, err
	}
	return ret, err
}

func (inr *Indexer) Update(entries []*IndexEntry) error {
	w := inr.store.BatchedWriter()
	for _, e := range entries {
		if e.Delete {
			w.Set(easyfl.Concat(e.AccountID, e.OutputID[:]), nil)
		} else {
			w.Set(easyfl.Concat(e.AccountID, e.OutputID[:]), []byte{0xff})
		}
	}
	return w.Commit()
}

// NewInMemory mostly for testing
func NewInMemory() *Indexer {
	return NewIndexer(common.NewInMemoryKVStore())
}
