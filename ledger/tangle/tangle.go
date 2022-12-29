package tangle

import (
	"crypto/ed25519"

	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraints"
	"github.com/lunfardo314/easyutxo/ledger/state"
	"github.com/lunfardo314/easyutxo/ledger/txbuilder"
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
		vertices          map[string]*Vertex
		state             ledger.StateStore
		milestoneOutputID ledger.OutputID
		finalStateRoot    common.VCommitment
	}
)

const milestoneDeposit = 1000

func InitMilestoneTransaction(initState state.Updatable, genesisPrivateKey ed25519.PrivateKey, finalityGadgetAddress constraints.AddressED25519) {
	genesisAddr := constraints.AddressED25519FromPrivateKey(genesisPrivateKey)

	genesisOutputBin, found := initState.Readable().GetUTXO(&ledger.GenesisOutputID)
	common.Assert(found, "can't find genesis output")
	genesisOutput, err := txbuilder.OutputFromBytes(genesisOutputBin)
	common.AssertNoError(err)
	totalSupply := genesisOutput.Amount()

}

// InitInMemoryTangle expects just initialized ledger state with the single genesis output
// It will create milestone for the genesis state
func InitInMemoryTangle(stateStore ledger.StateStore) *InMemoryTangle {

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
