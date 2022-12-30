package state

import (
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraints"
	"github.com/lunfardo314/unitrie/common"
	"golang.org/x/crypto/blake2b"
)

// Transaction provides access to the tree of transferable transaction
type Transaction struct {
	tree *lazyslice.Tree
	txid ledger.TransactionID
}

func MustTransactionFromTransferableBytes(txBytes []byte) *Transaction {
	ret := &Transaction{
		tree: lazyslice.TreeFromBytes(txBytes),
		txid: blake2b.Sum256(txBytes),
	}

	// validate what is possible without context

	easyfl.Assert(ret.tree.NumElements(Path(constraints.TxOutputs)) > 0, "MustTransactionFromTransferableBytes: number of outputs can't be 0")
	easyfl.Assert(ret.tree.NumElements(Path(constraints.TxInputIDs)) > 0, "MustTransactionFromTransferableBytes: number of inputs can't be 0")
	easyfl.Assert(ret.tree.NumElements(Path(constraints.TxEndorsements)) > constraints.MaxNumberOfEndorsements,
		"MustTransactionFromTransferableBytes: number of endorsements exceeds limit of %d", constraints.MaxNumberOfEndorsements)

	// check if inputs are unique
	inps := make(map[ledger.OutputID]struct{})
	ret.MustForEachInput(func(i byte, oid ledger.OutputID) bool {
		_, already := inps[oid]
		easyfl.Assert(!already, "MustTransactionFromTransferableBytes: repeating input @ %d", i)
		inps[oid] = struct{}{}
		return true
	})

	// check if endorsements are unique
	endorsements := make(map[ledger.TransactionID]struct{})
	ret.MustForEachEndorsement(func(i byte, txid ledger.TransactionID) bool {
		_, already := endorsements[txid]
		easyfl.Assert(!already, "MustTransactionFromTransferableBytes: repeating endorsement @ %d", i)
		endorsements[txid] = struct{}{}
		return true
	})
	return ret
}

func (tx *Transaction) ID() ledger.TransactionID {
	return tx.txid
}

func (tx *Transaction) NumProducedOutputs() int {
	return tx.tree.NumElements(Path(constraints.TxOutputs))
}

func (tx *Transaction) NumInputs() int {
	return tx.tree.NumElements(Path(constraints.TxInputIDs))
}

func (tx *Transaction) OutputAt(idx byte) []byte {
	return tx.tree.BytesAtPath(common.Concat(constraints.TxOutputs, idx))
}

func (tx *Transaction) MustForEachInput(fun func(i byte, oid ledger.OutputID) bool) {
	tx.tree.ForEach(func(i byte, data []byte) bool {
		oid, err := ledger.OutputIDFromBytes(data)
		common.Assert(err == nil, "MustForEachInput @ %d: %v", i, err)
		return fun(i, oid)
	}, Path(constraints.TxInputIDs))
}

func (tx *Transaction) MustForEachEndorsement(fun func(byte, ledger.TransactionID) bool) {
	tx.tree.ForEach(func(i byte, data []byte) bool {
		txid, err := ledger.TransactionIDFromBytes(data)
		common.Assert(err == nil, "MustForEachEndorsement @ %d: %v", i, err)
		return fun(i, txid)
	}, Path(constraints.TxEndorsements))
}

// MustForEachConsumedTransactionID iterates over unique transaction IDs consumed in
// the transaction in the order of appearance
func (tx *Transaction) MustForEachConsumedTransactionID(fun func(txid *ledger.TransactionID)) {
	already := make(map[ledger.TransactionID]struct{})
	tx.MustForEachInput(func(i byte, oid ledger.OutputID) bool {
		txid := oid.TransactionID()
		if _, found := already[txid]; !found {
			already[txid] = struct{}{}
			fun(&txid)
		}
		return true
	})
}

// FetchConsumedOutputs reads consumed output data from the ledger state
func (tx *Transaction) FetchConsumedOutputs(ledgerState ledger.StateReader) ([][]byte, error) {
	ret := make([][]byte, tx.tree.NumElements(Path(constraints.TxInputIDs)))
	var err error
	var found bool
	tx.MustForEachInput(func(i byte, oid ledger.OutputID) bool {
		if ret[i], found = ledgerState.GetUTXO(&oid); !found {
			err = fmt.Errorf("FetchConsumedOutputs:: can't find input %d,  %s of the transaction %s",
				i, oid.String(), tx.txid.String())
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}
