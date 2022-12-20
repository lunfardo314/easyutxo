package indexer

import (
	"fmt"
	"sync"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/library"
	"github.com/lunfardo314/unitrie/common"
)

type Indexer struct {
	mutex *sync.RWMutex
	store ledger.IndexerStore
}

// Command specifies update of 1 kv-pair in the indexer
type Command struct {
	ID        []byte
	OutputID  ledger.OutputID
	Delete    bool
	Partition Partition
}

type Partition byte

const (
	PartitionAccount = Partition(byte(iota))
	PartitionChainID
)

func (p Partition) String() string {
	switch p {
	case PartitionAccount:
		return "account"
	case PartitionChainID:
		return "chain"
	}
	return "unknown partition"
}

func (p Partition) Bytes() []byte {
	return []byte{byte(p)}
}

func (cmd *Command) String() string {
	return fmt.Sprintf("indexer command: partition: %s, id: %s, oid: %s, del: %v",
		cmd.Partition, easyfl.Fmt(cmd.ID[:]), cmd.OutputID.String(), cmd.Delete)
}

func NewIndexer(store ledger.IndexerStore, originAddr library.AddressED25519) *Indexer {
	w := store.BatchedWriter()
	var nullOutputID ledger.OutputID
	addrBytes := originAddr.Bytes()
	// account ID prefixed with length
	w.Set(easyfl.Concat(PartitionAccount, byte(len(addrBytes)), addrBytes, nullOutputID[:]), []byte{0xff})
	if err := w.Commit(); err != nil {
		panic(err)
	}
	return &Indexer{
		mutex: &sync.RWMutex{},
		store: store,
	}
}

func (inr *Indexer) GetUTXOsLockedInAccount(addr library.Accountable, state ledger.StateAccess) ([]*ledger.OutputDataWithID, error) {
	acc := addr.AccountID()
	if len(acc) > 255 {
		return nil, fmt.Errorf("accountID length should be <= 255")
	}
	accountPrefix := easyfl.Concat(PartitionAccount, byte(len(acc)), acc)

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

func (inr *Indexer) GetUTXOForChainID(id []byte, state ledger.StateAccess) (*ledger.OutputDataWithID, error) {
	if len(id) != 32 {
		return nil, fmt.Errorf("GetUTXOForChainID: chainID length must be 32-byte long")
	}
	key := easyfl.Concat(PartitionChainID, id)

	inr.mutex.RLock()
	defer inr.mutex.RUnlock()

	outID := inr.store.Get(key)
	if len(outID) == 0 {
		return nil, fmt.Errorf("GetUTXOForChainID: indexer record for chainID '%s' has not not been found", easyfl.Fmt(id))
	}
	oid, err := ledger.OutputIDFromBytes(outID)
	if err != nil {
		return nil, err
	}
	outData, found := state.GetUTXO(&oid)
	if !found {
		return nil, fmt.Errorf("GetUTXOForChainID: chain id: %s, outputID: %s. Output has not been found", easyfl.Fmt(id), oid)
	}
	return &ledger.OutputDataWithID{
		ID:         oid,
		OutputData: outData,
	}, nil
}

// accountID can be of different size, so it is prefixed with length

func (inr *Indexer) Update(commands []*Command) error {
	w := inr.store.BatchedWriter()
	for _, e := range commands {
		if err := e.run(w); err != nil {
			return err
		}
	}
	return w.Commit()
}

func (cmd *Command) run(w common.KVWriter) error {
	if len(cmd.ID) > 255 {
		return fmt.Errorf("indexer: ID length should be <= 255")
	}
	//fmt.Printf("+++++ %s\n", cmd.String())
	var key, value []byte
	switch cmd.Partition {
	case PartitionAccount:
		// ID is prefixed with length
		key = easyfl.Concat(PartitionAccount, byte(len(cmd.ID)), cmd.ID, cmd.OutputID[:])
		if !cmd.Delete {
			value = []byte{0xff}
		}

	case PartitionChainID:
		if len(cmd.ID) != 32 {
			return fmt.Errorf("indexer: chainID should be 32 bytes")
		}
		key = easyfl.Concat(PartitionChainID, cmd.ID)
		if !cmd.Delete {
			value = cmd.OutputID[:]
		}

	default:
		return fmt.Errorf("unsupported indeexer partition '%d'", cmd.Partition)
	}
	w.Set(key, value)
	return nil
}

// NewInMemory mostly for testing
func NewInMemory(originAddr library.AddressED25519) *Indexer {
	return NewIndexer(common.NewInMemoryKVStore(), originAddr)
}
