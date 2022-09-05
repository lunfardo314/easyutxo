package ledger

import (
	"encoding/hex"
	"fmt"

	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger/library"
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
	TxTreeIndexOutputGroups
	TxTreeIndexTimestamp
	TxTreeIndexContextCommitment
	TxTreeIndexLocalLibrary
	TxTreeIndexMax
)

// New empty ledger skeleton
func New() *Transaction {
	ret := &Transaction{tree: lazyslice.TreeEmpty()}
	ret.tree.PushEmptySubtrees(int(TxTreeIndexMax), nil)
	ret.tree.PushEmptySubtrees(1, Path(TxTreeIndexInputsLong))
	ret.tree.PushEmptySubtrees(1, Path(TxTreeIndexParamLong))
	ret.tree.PushEmptySubtrees(1, Path(TxTreeIndexOutputGroups))
	ret.tree.PushEmptySubtrees(1, Path(TxTreeIndexTimestamp))
	ret.tree.PushEmptySubtrees(1, Path(TxTreeIndexLocalLibrary))
	ret.tree.PushEmptySubtrees(1, Path(TxTreeIndexContextCommitment))
	return ret
}

// FromBytes ledger from bytes
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
	return tx.tree.NumElementsLong(Path(TxTreeIndexOutputGroups))
}

func (tx *Transaction) NumInputs() int {
	return tx.tree.NumElementsLong(Path(TxTreeIndexInputsLong))
}

func (tx *Transaction) CodeFromLocalLibrary(idx byte) []byte {
	return tx.tree.GetDataAtIdx(idx, Path(TxTreeIndexLocalLibrary))
}

func (tx *Transaction) ForEachOutputGroup(fun func(group byte) bool) {
	for i := 0; i < tx.tree.NumElements(Path(TxTreeIndexOutputGroups)); i++ {
		if !fun(byte(i)) {
			break
		}
	}
}

func (tx *Transaction) ForEachOutput(fun func(group, idx byte, o *Output) bool) {
	tx.tree.ForEachSubtree(func(group byte, stGroup *lazyslice.Tree) bool {
		var exit bool
		stGroup.ForEachSubtree(func(idx byte, stOutput *lazyslice.Tree) bool {
			exit = fun(group, idx, OutputFromTree(stOutput))
			return exit
		}, nil)
		return exit
	}, Path(TxTreeIndexOutputGroups))
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

func (tx *Transaction) Output(group, idx byte) *Output {
	return &Output{
		tree: lazyslice.TreeFromBytes(tx.tree.BytesAtPath(Path(TxTreeIndexOutputGroups, group, idx))),
	}
}

func (tx *Transaction) Validate() error {
	panic("implement me")
	//return easyutxo.CatchPanic(func() {
	//	tx.RunValidationScripts()
	//})
}

// CreateValidationContext finds all inputs in the ledger state.
// Creates a tree with ledger at long index 0 and all inputs at long index 1
//
func (tx *Transaction) CreateValidationContext(ledgerState LedgerState) (*ValidationContext, error) {
	ret := &ValidationContext{tree: lazyslice.TreeEmpty()}
	ret.tree.PushEmptySubtrees(int(ValidationCtxIndexMax), nil)
	ret.tree.PutSubtreeAtIdx(tx.Tree(), ValidationCtxTransactionIndex, nil)               // #0 ledger body
	ret.tree.PutSubtreeAtIdx(lazyslice.TreeEmpty(), ValidationCtxInputsIndex, nil)        // #1 validation context (inputs)
	ret.tree.PutSubtreeAtIdx(library.ScriptLibrary, ValidationCtxGlobalLibraryIndex, nil) // #2 global script library tree

	var err error
	tx.ForEachInput(func(idx uint16, oid OutputID) bool {
		od, ok := ledgerState.GetUTXO(&oid)
		if !ok {
			err = fmt.Errorf("input not found: %s", oid.String())
			return false
		}
		ret.tree.PushLongAtPath(od, Path(ValidationCtxInputsIndex))
		return true
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}
