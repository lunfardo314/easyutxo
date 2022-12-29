package ledger

import (
	"errors"
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger/constraints"
	"github.com/lunfardo314/unitrie/common"
	"github.com/lunfardo314/unitrie/models/trie_blake2b"
)

const (
	TransactionIDLength = 32
	OutputIDLength      = TransactionIDLength + 1
)

var CommitmentModel = trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256)

// GenesisOutputID is an all0 outputID
var GenesisOutputID OutputID

type (
	TransactionID [TransactionIDLength]byte
	OutputID      [OutputIDLength]byte

	OutputDataWithID struct {
		ID         OutputID
		OutputData []byte
	}

	OutputDataWithChainID struct {
		OutputDataWithID
		ChainID                    [32]byte
		PredecessorConstraintIndex byte
	}

	StateReadAccess interface {
		GetUTXO(id *OutputID) ([]byte, bool)
		HasTransaction(txid *TransactionID) bool
	}

	IndexerAccess interface {
		GetUTXOsLockedInAccount(accountID constraints.Accountable, state StateReadAccess) ([]*OutputDataWithID, error)
		GetUTXOForChainID(id []byte, state StateReadAccess) (*OutputDataWithID, error)
	}

	StateStore interface {
		common.KVReader
		common.BatchedUpdatable
	}

	IndexerStore interface {
		common.BatchedUpdatable
		common.Traversable
		common.KVReader
	}
)

func TransactionIDFromBytes(data []byte) (ret TransactionID, err error) {
	if len(data) != TransactionIDLength {
		err = errors.New("TransactionIDFromBytes: wrong data length")
		return
	}
	copy(ret[:], data)
	return
}

func (txid *TransactionID) Bytes() []byte {
	return txid[:]
}

func (txid *TransactionID) String() string {
	return easyfl.Fmt(txid[:])
}

func NewOutputID(id TransactionID, idx byte) (ret OutputID) {
	copy(ret[:TransactionIDLength], id[:])
	ret[TransactionIDLength] = idx
	return
}

func OutputIDFromBytes(data []byte) (ret OutputID, err error) {
	if len(data) != OutputIDLength {
		err = errors.New("OutputIDFromBytes: wrong data length")
		return
	}
	copy(ret[:], data)
	return
}

func (oid *OutputID) String() string {
	txid := oid.TransactionID()
	return fmt.Sprintf("[%d]%s", oid.Index(), txid.String())
}

func (oid *OutputID) TransactionID() (ret TransactionID) {
	copy(ret[:], oid[:TransactionIDLength])
	return
}

func (oid *OutputID) Index() byte {
	return oid[TransactionIDLength]
}

func (oid *OutputID) Bytes() []byte {
	return oid[:]
}
