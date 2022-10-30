package indexer

import (
	"fmt"
	"sync"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraint"
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

func NewIndexer(store ledger.IndexerStore, originAddr constraint.AddressED25519) *Indexer {
	w := store.BatchedWriter()
	var nullOutputID ledger.OutputID
	addrBytes := originAddr.Bytes()
	// account ID prefixed with length
	w.Set(easyfl.Concat(byte(len(addrBytes)), addrBytes, nullOutputID[:]), []byte{0xff})
	if err := w.Commit(); err != nil {
		panic(err)
	}
	return &Indexer{
		mutex: &sync.RWMutex{},
		store: store,
	}
}

func (inr *Indexer) GetUTXOsForAccountID(addr constraint.Accountable, state ledger.StateAccess) ([]*ledger.OutputDataWithID, error) {
	acc := addr.AccountID()
	if len(acc) > 255 {
		return nil, fmt.Errorf("accountID length should be <= 255")
	}
	accountPrefix := easyfl.Concat(byte(len(acc)), acc)

	inr.mutex.RLock()
	defer inr.mutex.RUnlock()

	ret := make([]*ledger.OutputDataWithID, 0)
	var err error
	var found bool
	inr.store.Iterator(accountPrefix).Iterate(func(k, v []byte) bool {
		o := &ledger.OutputDataWithID{}
		o.ID, err = ledger.OutputIDFromBytes(k[len(accountPrefix):])
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

// accountID can be of different size, so it is prefixed with length

func (inr *Indexer) Update(entries []*IndexEntry) error {
	w := inr.store.BatchedWriter()
	for _, e := range entries {
		if len(e.AccountID) > 255 {
			return fmt.Errorf("accountID length should be <= 255")
		}
		// accountID is prefixed with length
		key := easyfl.Concat(byte(len(e.AccountID)), e.AccountID, e.OutputID[:])
		if e.Delete {
			w.Set(key, nil)
		} else {
			w.Set(key, []byte{0xff})
		}
	}
	return w.Commit()
}

// NewInMemory mostly for testing
func NewInMemory(originAddr constraint.AddressED25519) *Indexer {
	return NewIndexer(common.NewInMemoryKVStore(), originAddr)
}
