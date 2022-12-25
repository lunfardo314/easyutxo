package tangle

import (
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/unitrie/common"
)

type (
	Access interface {
		GetVertex(txid *ledger.TransactionID) (*Vertex, bool)
		HasVertex(txid *ledger.TransactionID) bool
	}
	Vertex struct {
		txBytes   []byte
		stateRoot common.VCommitment
	}

	InMemoryTangle struct {
		vertices map[string]*Vertex
		state    ledger.StateStore
	}
)

func NewInMemoryTangle(state ledger.StateStore) *InMemoryTangle {
	return &InMemoryTangle{
		vertices: make(map[string]*Vertex),
		state:    state,
	}
}

func (t InMemoryTangle) GetVertex(txid *ledger.TransactionID) (*Vertex, bool) {
	ret, ok := t.vertices[string((*txid)[:])]
	return ret, ok
}

func (t InMemoryTangle) HasVertex(txid *ledger.TransactionID) bool {
	_, ok := t.vertices[string((*txid)[:])]
	return ok
}
