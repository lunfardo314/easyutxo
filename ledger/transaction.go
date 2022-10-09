package ledger

import (
	"encoding/hex"

	"github.com/lunfardo314/easyutxo/lazyslice"
	"golang.org/x/crypto/blake2b"
)

const (
	TransactionIDLength = 32
)

type TransactionID []byte

func (txid TransactionID) String() string {
	return hex.EncodeToString(txid)
}

type Transaction struct {
	tree *lazyslice.Tree
}

// NewTransaction empty transaction blocks skeleton
func NewTransaction() *Transaction {
	ret := &Transaction{tree: lazyslice.TreeEmpty()}
	ret.tree.PushEmptySubtrees(int(TxTreeIndexMax), nil)
	ret.tree.PutSubtreeAtIdx(lazyslice.TreeEmpty(), TxUnlockParamsBranch, nil)
	ret.tree.PutSubtreeAtIdx(lazyslice.TreeEmpty(), TxInputIDsBranch, nil)
	ret.tree.PutSubtreeAtIdx(lazyslice.TreeEmpty(), TxOutputBranch, nil)
	ret.tree.PutDataAtIdx(TxTimestamp, nil, nil)
	ret.tree.PutDataAtIdx(TxInputCommitment, nil, nil)
	ret.tree.PutSubtreeAtIdx(lazyslice.TreeEmpty(), TxLocalLibraryBranch, nil)
	return ret
}

// TransactionFromBytes ledger from bytes
func TransactionFromBytes(data []byte) *Transaction {
	return &Transaction{tree: lazyslice.TreeFromBytes(data)}
}

func (tx *Transaction) Tree() *lazyslice.Tree {
	return tx.tree
}

func (tx *Transaction) Bytes() []byte {
	return tx.tree.Bytes()
}

func (tx *Transaction) ID() TransactionID {
	ret := blake2b.Sum256(tx.Bytes())
	return ret[:]
}

func (tx *Transaction) NumOutputs() int {
	ret := 0
	tx.tree.ForEachSubtree(func(idx byte, subtree *lazyslice.Tree) bool {
		ret += subtree.NumElements(nil)
		return true
	}, Path(TxOutputBranch))
	return ret
}

func (tx *Transaction) NumInputs() int {
	return tx.tree.NumElements(Path(TxInputIDsBranch))
}

func (tx *Transaction) CodeFromLocalLibrary(idx byte) []byte {
	return tx.tree.BytesAtPath(Path(TransactionBranch, TxLocalLibraryBranch, idx))
}

func (tx *Transaction) ForEachOutput(fun func(o *Output, idx byte) bool) {
	tx.tree.ForEach(func(idx byte, outputData []byte) bool {
		return fun(OutputFromBytes(outputData), idx)
	}, Path(TxOutputBranch))
}

func (tx *Transaction) ForEachInputID(fun func(idx byte, o OutputID) bool) {
	var oid OutputID
	var err error
	tx.tree.ForEach(func(i byte, data []byte) bool {
		if oid, err = OutputIDFromBytes(data); err != nil {
			panic(err)
		}
		return !fun(i, oid)
	}, Path(TransactionBranch, TxInputIDsBranch))
}
