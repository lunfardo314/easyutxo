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
	vertex := NewVertex(tx)
	transactionSolidityStatus, allSolid := tg.solidityOfInputTransactions(tx)

	// fetch all consumed UTXO what is possible. Check if transaction does not
	// trie to consume already spent or non-existent UTXOs
	consumed := make([][]byte, tx.NumInputs())
	transactionValid := true
	tx.MustForEachInput(func(i byte, oid *ledger.OutputID) bool {
		txid := oid.TransactionID()
		switch transactionSolidityStatus[txid] {
		case transactionSolidityFinal:
			out, found := tg.finalStateReader.GetUTXO(oid)
			if !found {
				// if UTXO is not on final state, it means it is spent or never existed
				// Transaction is invalid
				transactionValid = false
				return false
			}
			consumed[i] = out
		case transactionSolidityOnTangle:
			v, found := tg.GetVertex(&txid)
			common.Assert(found, "AddTransaction: expected on the tangle txid = %s", txid.String())
			if int(oid.Index()) >= v.tx.NumProducedOutputs() {
				// transaction os ok, but output index is wrong
				// Transaction is invalid
				transactionValid = true
				return false
			}
			// if we find UTXO on the tangle, it is spendable, but may be a double spend
			consumed[i] = v.tx.OutputAt(oid.Index())
		}
		if consumed[i] != nil {
			isDoubleSpend := tg.registerSpentOutput(oid)
			if isDoubleSpend {
				// TODO maintain conflict sets and double spend counts
			}
		}
		return true
	})
	if !transactionValid {
		// some outputs are spent or invalid
		return false
	}
	if !allSolid {
		// some inputs not solid
		for txid, status := range transactionSolidityStatus {
			if status == transactionSolidityNotSolid {
				tg.RequireSolidification(vertex, txid)
			}
		}
		return false
	}

	return !transactionValid
}

// registerSpentOutput keeps register of spend outputs. returns if it is double spend
func (tg *InMemoryTangle) registerSpentOutput(oid *ledger.OutputID) bool {
	panic("not implemented")
}

func (tg *InMemoryTangle) solidityOfInputTransactions(tx *state.Transaction) (map[ledger.TransactionID]byte, bool) {
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
	tx.MustForEachEndorsement(func(_ byte, txid ledger.TransactionID) bool {
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

func (tg *InMemoryTangle) RequireSolidification(vertex *Vertex, txidChild ledger.TransactionID) {
	panic("RequireSolidification: implement me")
}
