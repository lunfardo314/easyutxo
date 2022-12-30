package tangle

import (
	"sync"

	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/state"
	"github.com/lunfardo314/unitrie/common"
)

type (
	Access interface {
		Reader() ledger.StateReader
		GetVertex(*ledger.TransactionID) (*Vertex, bool)
		HasVertex(*ledger.TransactionID) bool
	}

	Vertex struct {
		mutex     sync.RWMutex // do not copy!
		tx        *state.Transaction
		stateRoot common.VCommitment
	}

	InMemoryTangle struct {
		mutex             sync.RWMutex
		stateStore        ledger.StateStore
		vertices          map[ledger.TransactionID]*Vertex
		milestoneOutputID ledger.OutputID
		finalStateReader  *state.Readable
	}
)

const stateReaderCacheClearAt = 1000

// NewInMemoryTangle expects just initialized ledger state with the
// genesis output and genesis milestone output
func NewInMemoryTangle(stateStore ledger.StateStore, milestoneOutputID ledger.OutputID, stateRoot common.VCommitment) *InMemoryTangle {
	stateReader, err := state.NewReadable(stateStore, stateRoot, stateReaderCacheClearAt)
	common.AssertNoError(err)
	return &InMemoryTangle{
		vertices:          make(map[ledger.TransactionID]*Vertex),
		stateStore:        stateStore,
		milestoneOutputID: milestoneOutputID,
		finalStateReader:  stateReader,
	}
}

func NewVertex(tx *state.Transaction) *Vertex {
	return &Vertex{
		mutex:     sync.RWMutex{},
		tx:        tx,
		stateRoot: nil,
	}
}

func (v *Vertex) Transaction() *state.Transaction {
	return v.tx
}

func (tg *InMemoryTangle) GetVertex(txid *ledger.TransactionID) (*Vertex, bool) {
	tg.mutex.RLock()
	defer tg.mutex.RUnlock()

	ret, ok := tg.vertices[*txid]
	return ret, ok
}

func (tg *InMemoryTangle) HasVertex(txid *ledger.TransactionID) bool {
	tg.mutex.RLock()
	defer tg.mutex.RUnlock()

	_, ok := tg.vertices[*txid]
	return ok
}

func (tg *InMemoryTangle) Reader() ledger.StateReader {
	return tg
}

func (tg *InMemoryTangle) GetUTXO(oid *ledger.OutputID) ([]byte, bool) {
	txid := oid.TransactionID()
	v, found := tg.GetVertex(&txid)
	if found {
		// output belongs to the transaction in the tangle
		idx := oid.Index()
		if int(idx) >= v.tx.NumProducedOutputs() {
			return nil, false
		}
		return v.tx.OutputAt(idx), true
	}
	// looking for the output in the final state
	return tg.finalStateReader.GetUTXO(oid)
}

func (tg *InMemoryTangle) HasTransaction(txid *ledger.TransactionID) bool {
	if tg.HasVertex(txid) {
		return true
	}
	return tg.finalStateReader.HasTransaction(txid)
}
