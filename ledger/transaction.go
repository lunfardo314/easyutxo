package ledger

import (
	"encoding/hex"

	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger/globalpath"
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

// New empty transaction tree skeleton
func New() *Transaction {
	ret := &Transaction{tree: lazyslice.TreeEmpty()}
	ret.tree.PushEmptySubtrees(int(globalpath.TxTreeIndexMax), nil)
	ret.tree.PushEmptySubtrees(1, globalpath.TxUnlockParamsLong)
	ret.tree.PushEmptySubtrees(1, globalpath.TxInputIDsLong)
	ret.tree.PushEmptySubtrees(1, globalpath.TxOutputGroups)
	ret.tree.PushEmptySubtrees(1, globalpath.TxTimestamp)
	ret.tree.PushEmptySubtrees(1, globalpath.TxLocalLibrary)
	ret.tree.PushEmptySubtrees(1, globalpath.TxConsumedInputCommitment)
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
	ret := 0
	tx.tree.ForEachSubtree(func(idx byte, subtree *lazyslice.Tree) bool {
		ret += subtree.NumElements(nil)
		return true
	}, globalpath.TxOutputGroups)
	return ret
}

func (tx *Transaction) NumInputs() int {
	return tx.tree.NumElementsLong(globalpath.TxInputIDsLong)
}

func (tx *Transaction) CodeFromLocalLibrary(idx byte) []byte {
	return tx.tree.GetDataAtIdx(idx, globalpath.TxLocalLibrary)
}

func (tx *Transaction) ForEachOutputGroup(fun func(group byte) bool) {
	for i := 0; i < tx.tree.NumElements(globalpath.TxOutputGroups); i++ {
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
	}, globalpath.TxOutputGroups)
}

func (tx *Transaction) ForEachInputID(fun func(idx uint16, o OutputID) bool) {
	var oid OutputID
	var err error
	for i := 0; i < tx.NumInputs(); i++ {
		d := tx.tree.GetBytesAtIdxLong(uint16(i), globalpath.TxInputIDsLong)
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
		tree: lazyslice.TreeFromBytes(tx.tree.BytesAtPath(lazyslice.PathMakeAppend(globalpath.TxOutputGroups, group, idx))),
	}
}

func (tx *Transaction) Validate() error {
	panic("implement me")
	//return easyutxo.CatchPanic(func() {
	//	tx.RunValidationScripts()
	//})
}
