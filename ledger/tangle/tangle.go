package tangle

import (
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/unitrie/common"
)

type (
	Access interface {
		GetConeTip(txid *ledger.TransactionID) (*ConeTip, bool)
		HasConeTip(txid *ledger.TransactionID) bool
	}
	ConeTip struct {
		txBytes   []byte
		stateRoot common.VCommitment
	}

	InMemoryTangle struct {
		vertices map[string]*ConeTip
		state    ledger.StateStore
	}
)

func NewInMemoryTangle(state ledger.StateStore) *InMemoryTangle {
	return &InMemoryTangle{
		vertices: make(map[string]*ConeTip),
		state:    state,
	}
}

func (t InMemoryTangle) GetConeTip(txid *ledger.TransactionID) (*ConeTip, bool) {
	ret, ok := t.vertices[string((*txid)[:])]
	return ret, ok
}

func (t InMemoryTangle) HasConeTip(txid *ledger.TransactionID) bool {
	_, ok := t.vertices[string((*txid)[:])]
	return ok
}
