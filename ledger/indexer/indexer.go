package indexer

import (
	"fmt"
	"sync"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/library"
)

type Indexer struct {
	mutex *sync.RWMutex
	store ledger.IndexerStore
}

type IndexEntry struct {
	ID        []byte
	OutputID  ledger.OutputID
	Delete    bool
	Partition byte
}

const (
	IndexedPartitionAccount = byte(iota)
	IndexedPartitionChainID
)

func NewIndexer(store ledger.IndexerStore, originAddr library.AddressED25519) *Indexer {
	w := store.BatchedWriter()
	var nullOutputID ledger.OutputID
	addrBytes := originAddr.Bytes()
	// account ID prefixed with length
	w.Set(easyfl.Concat(IndexedPartitionAccount, byte(len(addrBytes)), addrBytes, nullOutputID[:]), []byte{0xff})
	if err := w.Commit(); err != nil {
		panic(err)
	}
	return &Indexer{
		mutex: &sync.RWMutex{},
		store: store,
	}
}

func (inr *Indexer) GetUTXOsForAccountID(addr library.Accountable, state ledger.StateAccess) ([]*ledger.OutputDataWithID, error) {
	acc := addr.AccountID()
	if len(acc) > 255 {
		return nil, fmt.Errorf("accountID length should be <= 255")
	}
	accountPrefix := easyfl.Concat(IndexedPartitionAccount, byte(len(acc)), acc)

	inr.mutex.RLock()
	defer inr.mutex.RUnlock()

	ret := make([]*ledger.OutputDataWithID, 0)
	var err error
	var found bool
	inr.store.Iterator(accountPrefix).Iterate(func(k, _ []byte) bool {
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

// TODO refactor indexing of chainIDs

func (inr *Indexer) GetUTXOForChainID(id []byte, state ledger.StateAccess) (*ledger.OutputDataWithID, error) {
	if len(id) != 32 {
		return nil, fmt.Errorf("chainID length must be 32-byte long")
	}
	var ret *ledger.OutputDataWithID
	err := fmt.Errorf("can't find indexed output with chain id %s", easyfl.Fmt(id))
	accountPrefix := easyfl.Concat(IndexedPartitionChainID, byte(32), id)
	inr.store.Iterator(accountPrefix).Iterate(func(k, _ []byte) bool {
		ret = &ledger.OutputDataWithID{
			ID:         ledger.OutputID{},
			OutputData: nil,
		}
		ret.ID, err = ledger.OutputIDFromBytes(k[len(accountPrefix):])
		if err != nil {
			return false
		}
		var found bool
		ret.OutputData, found = state.GetUTXO(&ret.ID)
		if found {
			err = nil
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// accountID can be of different size, so it is prefixed with length

func (inr *Indexer) Update(entries []*IndexEntry) error {
	w := inr.store.BatchedWriter()
	for _, e := range entries {
		if len(e.ID) > 255 {
			return fmt.Errorf("indexer ID length should be <= 255")
		}
		// ID is prefixed with length
		key := easyfl.Concat(e.Partition, byte(len(e.ID)), e.ID, e.OutputID[:])
		if e.Delete {
			w.Set(key, nil)
		} else {
			w.Set(key, []byte{0xff})
		}
	}
	return w.Commit()
}

// NewInMemory mostly for testing
func NewInMemory(originAddr library.AddressED25519) *Indexer {
	return NewIndexer(common.NewInMemoryKVStore(), originAddr)
}
