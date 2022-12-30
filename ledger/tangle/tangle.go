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
		WaitForSolidification(txid ledger.TransactionID)
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

func (v *Vertex) StateReader(store common.KVReader) (*state.Readable, error) {
	return state.NewReadable(store, v.stateRoot)
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

const (
	transactionSolidityOnTangle = byte(iota)
	transactionSolidityFinal
	transactionSolidityNotSolid
)

func (tg *InMemoryTangle) AddTransaction(tx *state.Transaction) bool {
	transactionSolidityStatus, allSolid := tg.inputSolidityStatus(tx)
	if !allSolid {
		for txid := range transactionSolidityStatus {
			tg.WaitForSolidification(txid)
		}
		return false
	}
	consumed := make([][]byte, tx.NumInputs())
	alreadySpent := false
	tx.MustForEachInput(func(i byte, oid *ledger.OutputID) bool {
		txid := oid.TransactionID()
		switch transactionSolidityStatus[txid] {
		case transactionSolidityFinal:
			out, found := tg.finalStateReader.GetUTXO(oid)
			if !found {
				alreadySpent = true
				return false
			}
			consumed[i] = out
		case transactionSolidityOnTangle:
			v, found := tg.GetVertex(&txid)
			common.Assert(found, "AddTransaction: inconsistency 1")
			rdr, err := v.StateReader(tg.stateStore)
			common.AssertNoError(err)
			consumed[i], found = rdr.GetUTXO(oid)
			if !found {
				alreadySpent = true
				return false
			}

		default:
			panic("AddTransaction: inconsistency 2")
		}
		return true
	})
	// TODO
	return !alreadySpent
}

func (tg *InMemoryTangle) inputSolidityStatus(tx *state.Transaction) (map[ledger.TransactionID]byte, bool) {
	ret := make(map[ledger.TransactionID]byte)
	allSolid := true
	tx.MustForEachInput(func(_ byte, oid *ledger.OutputID) bool {
		txid := oid.TransactionID()
		if _, ok := ret[txid]; ok {
			return true
		}
		if tg.HasVertex(&txid) {
			ret[txid] = transactionSolidityOnTangle
			return true
		}
		if tg.finalStateReader.HasTransaction(&txid) {
			ret[txid] = transactionSolidityFinal
			return true
		}
		ret[txid] = transactionSolidityNotSolid
		allSolid = false
		return true
	})
	return ret, allSolid
}

func (tg *InMemoryTangle) WaitForSolidification(txid ledger.TransactionID) {
	panic("WaitForSolidification: implement me")
}
