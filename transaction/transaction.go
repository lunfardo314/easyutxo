package transaction

import (
	"encoding/hex"
	"fmt"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"golang.org/x/crypto/blake2b"
)

const (
	IDLength = 32
)

type ID [IDLength]byte

func (txid *ID) String() string {
	return hex.EncodeToString(txid[:])
}

var Path = lazyslice.Path

type Transaction struct {
	tree *lazyslice.Tree
}

const (
	TxTreeIndexInputsLong = byte(iota)
	TxTreeIndexParamLong
	TxTreeIndexOutputsLong
	TxTreeIndexTimestamp
	TxTreeIndexContextCommitment
	TxTreeIndexMax
)

// New empty transaction skeleton
func New() *Transaction {
	ret := &Transaction{tree: lazyslice.TreeEmpty()}
	ret.tree.PushEmptySubtrees(int(TxTreeIndexMax), nil)
	ret.tree.PushEmptySubtrees(1, Path(TxTreeIndexInputsLong))
	ret.tree.PushEmptySubtrees(1, Path(TxTreeIndexParamLong))
	ret.tree.PushEmptySubtrees(1, Path(TxTreeIndexOutputsLong))
	ret.tree.PushEmptySubtrees(1, Path(TxTreeIndexTimestamp))
	ret.tree.PushEmptySubtrees(1, Path(TxTreeIndexContextCommitment))
	return ret
}

// FromBytes transaction from bytes
func FromBytes(data []byte) *Transaction {
	return &Transaction{tree: lazyslice.TreeFromBytes(data)}
}

func (tx *Transaction) Tree() *lazyslice.Tree {
	return tx.tree
}

func (tx *Transaction) Bytes() []byte {
	return tx.tree.Bytes()
}

func (tx *Transaction) ID() ID {
	return blake2b.Sum256(tx.Bytes())
}

func (tx *Transaction) NumOutputs() int {
	return tx.tree.NumElementsLong(Path(TxTreeIndexOutputsLong))
}

func (tx *Transaction) NumInputs() int {
	return tx.tree.NumElementsLong(Path(TxTreeIndexInputsLong))
}

func (tx *Transaction) ForEachOutput(fun func(idx uint16, o *Output) bool) {
	for i := 0; i < tx.NumOutputs(); i++ {
		d := tx.tree.GetBytesAtIdxLong(uint16(i), Path(TxTreeIndexOutputsLong))
		if !fun(uint16(i), OutputFromBytes(d)) {
			break
		}
	}
}

func (tx *Transaction) ForEachInput(fun func(idx uint16, o OutputID) bool) {
	var oid OutputID
	var err error
	for i := 0; i < tx.NumInputs(); i++ {
		d := tx.tree.GetBytesAtIdxLong(uint16(i), Path(TxTreeIndexInputsLong))
		if oid, err = OutputIDFromBytes(d); err != nil {
			panic(err)
		}
		if !fun(uint16(i), oid) {
			break
		}
	}
}

func (tx *Transaction) Output(outputContext, idx byte) *Output {
	return &Output{
		tree: lazyslice.TreeFromBytes(tx.tree.BytesAtPath(Path(TxTreeIndexOutputsLong, outputContext, idx))),
	}
}

func (tx *Transaction) Validate() error {
	return easyutxo.CatchPanic(func() {
		tx.RunValidationScripts()
	})
}

func (tx *Transaction) RunValidationScripts() {

}

func (tx *Transaction) CheckUnboundedConstraints() {

}

// CreateValidationContext finds all inputs in the ledger state.
// Creates a tree with transaction at long index 0 and all inputs at long index 1
func (tx *Transaction) CreateValidationContext(ledgerState LedgerState) (*ValidationContext, error) {
	ret := &ValidationContext{tree: lazyslice.TreeEmpty()}
	ret.tree.PushEmptySubtrees(int(ValidationCtxIndexMax), nil)
	ret.tree.PutSubtreeAtIdx(tx.Tree(), ValidationCtxTxIndex, nil)                  // #0 transaction body
	ret.tree.PutSubtreeAtIdx(lazyslice.TreeEmpty(), ValidationCtxtInputsIndex, nil) // #1 validation context (inputs)
	ret.tree.PutSubtreeAtIdx(ScriptLibrary, ValidationCtxScriptLibraryIndex, nil)   // #2 script library tree

	var err error
	txid := tx.ID()
	tx.ForEachInput(func(idx uint16, oid OutputID) bool {
		outputID := NewOutputID(txid, idx)
		od, ok := ledgerState.GetUTXO(&outputID)
		if !ok {
			err = fmt.Errorf("input not found: %s", oid.String())
			return false
		}
		ret.tree.PushLongAtPath(od, Path(ValidationCtxtInputsIndex))
		return true
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}
