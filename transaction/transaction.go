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

type Transaction struct {
	tree *lazyslice.Tree
}

type ValidationContext struct {
	tree *lazyslice.Tree
}

const (
	TxTreeIndexInputsLong      = byte(0)
	TxTreeIndexParamLong       = byte(1)
	TxTreeIndexOutputsLong     = byte(2)
	TxTreeIndexTimestamp       = byte(3)
	TxTreeIndexInputCommitment = byte(4)

	ValidationContextTxIndex     = byte(0)
	ValidationContextInputsIndex = byte(1)
)

// New empty transaction skeleton
func New() *Transaction {
	ret := &Transaction{tree: lazyslice.TreeEmpty()}
	ret.tree.PushEmptySubtreeAtPath() // #0 inputs long
	ret.tree.PushEmptySubtreeAtPath() // #1 params long
	ret.tree.PushEmptySubtreeAtPath() // #2 outputs long
	return ret
}

// FromBytes transaction from bytes
func FromBytes(data []byte) *Transaction {
	return &Transaction{tree: lazyslice.TreeFromBytes(data)}
}

func (tx *Transaction) Bytes() []byte {
	return tx.tree.Bytes()
}

func (tx *Transaction) ID() ID {
	return blake2b.Sum256(tx.Bytes())
}

func (tx *Transaction) NumOutputs() int {
	return tx.tree.NumElementsLong(TxTreeIndexOutputsLong)
}

func (tx *Transaction) NumInputs() int {
	return tx.tree.NumElementsLong(TxTreeIndexInputsLong)
}

func (tx *Transaction) ForEachOutput(fun func(idx uint16, o *Output) bool) {
	for i := 0; i < tx.NumOutputs(); i++ {
		d := tx.tree.GetDataAtIdxLong(uint16(i), TxTreeIndexOutputsLong)
		if !fun(uint16(i), OutputFromBytes(d)) {
			break
		}
	}
}

func (tx *Transaction) ForEachInput(fun func(idx uint16, o OutputID) bool) {
	var oid OutputID
	var err error
	for i := 0; i < tx.NumInputs(); i++ {
		d := tx.tree.GetDataAtIdxLong(uint16(i), TxTreeIndexInputsLong)
		if oid, err = OutputIDFromBytes(d); err != nil {
			panic(err)
		}
		if !fun(uint16(i), oid) {
			break
		}
	}
}

// GetValidationContext finds all inputs in the ledger state and pushes their data
// into the validation context at long index
func (tx *Transaction) GetValidationContext(ledgerState LedgerState) (*ValidationContext, error) {
	var err error
	txid := tx.ID()
	validationCtx := lazyslice.TreeEmpty()
	tx.ForEachInput(func(idx uint16, oid OutputID) bool {
		outputID := NewOutputID(txid, idx)
		od, ok := ledgerState.GetUTXO(&outputID)
		if !ok {
			err = fmt.Errorf("input not found: %s", oid.String())
			return false
		}
		validationCtx.PushLong(od, ValidationContextInputsIndex)
		return true
	})
	if err != nil {
		return nil, err
	}
	ret := &ValidationContext{tree: lazyslice.TreeEmpty()}
	ret.tree.PushSubtreeFromBytesAtPath(tx.Bytes())            // #0 transaction body
	ret.tree.PushSubtreeFromBytesAtPath(validationCtx.Bytes()) // #1 validation context (inputs)

	return ret, nil
}

func (v *ValidationContext) Transaction() *Transaction {
	return FromBytes(v.tree.BytesAtPath(0))
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
