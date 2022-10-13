package ledger

import (
	ed255192 "crypto/ed25519"
	"encoding/binary"
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

// TransactionContext is a data structure, which contains transaction, consumed outputs and constraint library
type TransactionContext struct {
	tree *lazyslice.Tree
}

var Path = lazyslice.Path

// Top level banches
const (
	TransactionBranch = byte(iota)
	ConsumedContextBranch
)

// ConsumedContextBranch. 1st level branches
const (
	ConsumedContextOutputsBranch = byte(iota)
	ConsumedContextConstraintLibraryBranch
)

// TransactionBranch. 1st level branches
const (
	TxUnlockParamsBranch = byte(iota)
	TxInputIDsBranch
	TxOutputBranch
	TxTimestamp
	TxInputCommitment
	TxLocalLibraryBranch
	TxTreeIndexMax
)

// Invocation types are indices of constraints in the global library
const (
	InvocationTypeInline = byte(iota)
	InvocationTypeLocalLib
	InvocationTypeFirstGlobal
)

func NewTransactionContext() *TransactionContext {
	tx := NewTransaction()
	ret, err := TransactionContextFromTransaction(tx.Bytes(), nil)
	easyutxo.AssertNoError(err)
	return ret
}

// TransactionContextFromTransaction finds all inputs in the ledger state.
// Creates a blocks with ledger at long index 0 and all inputs at long index 1
func TransactionContextFromTransaction(txBytes []byte, ledgerState LedgerState) (*TransactionContext, error) {
	tx := TransactionFromBytes(txBytes)
	ret := &TransactionContext{tree: lazyslice.TreeEmpty()}
	ret.tree.PushEmptySubtrees(2, nil)
	ret.tree.PutSubtreeAtIdx(tx.Tree(), TransactionBranch, nil)                                                   // #0 transaction
	ret.tree.PushEmptySubtrees(2, Path(ConsumedContextBranch))                                                    // #1 consumed context (inputs, library)
	ret.tree.PutSubtreeAtIdx(lazyslice.TreeEmpty(), ConsumedContextOutputsBranch, Path(ConsumedContextBranch))    // 1 @ 0 consumed outputs
	ret.tree.PutSubtreeAtIdx(constraintTree, ConsumedContextConstraintLibraryBranch, Path(ConsumedContextBranch)) // 1 @ 1 script library blocks

	var err error
	consPath := Path(ConsumedContextBranch, ConsumedContextOutputsBranch)
	var oid OutputID

	ret.tree.ForEach(func(i byte, data []byte) bool {
		if oid, err = OutputIDFromBytes(data); err != nil {
			return false
		}
		od, ok := ledgerState.GetUTXO(&oid)
		if !ok {
			err = fmt.Errorf("input not found: %s", oid.String())
			return false
		}
		ret.tree.PushData(od, consPath)
		return true
	}, Path(TransactionBranch, TxOutputBranch))

	if err != nil {
		return nil, err
	}
	return ret, nil
}

func TransactionContextFromTree(dataTree *lazyslice.Tree) *TransactionContext {
	return &TransactionContext{
		tree: dataTree,
	}
}

func invokeConstraint(tree *lazyslice.Tree, path lazyslice.TreePath) []byte {
	return TransactionContextFromTree(tree).Invoke(path)
}

func (v *TransactionContext) Tree() *lazyslice.Tree {
	return v.tree
}

func (v *TransactionContext) Transaction() *Transaction {
	return TransactionFromBytes(v.tree.BytesAtPath(Path(TransactionBranch)))
}

func (v *TransactionContext) CodeFromGlobalLibrary(idx byte) []byte {
	return v.tree.BytesAtPath(Path(ConsumedContextBranch, ConsumedContextConstraintLibraryBranch, idx))
}

func (v *TransactionContext) CodeFromLocalLibrary(idx byte) []byte {
	return v.tree.BytesAtPath(Path(TransactionBranch, TxLocalLibraryBranch, idx))
}

func (v *TransactionContext) parseInvocationCode(invocationFullPath lazyslice.TreePath) []byte {
	invocation := v.tree.BytesAtPath(invocationFullPath)
	if len(invocation) < 1 {
		panic("empty invocation")
	}
	switch invocation[0] {
	case InvocationTypeLocalLib:
		if len(invocation) < 2 {
			panic("wrong invocation")
		}
		return v.CodeFromLocalLibrary(invocation[1])
	case InvocationTypeInline:
		return invocation[1:]
	}
	return v.CodeFromGlobalLibrary(invocation[0])
}

func (v *TransactionContext) rootContext() *DataContext {
	return NewDataContext(v.tree, nil)
}

func (v *TransactionContext) TransactionBytes() []byte {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txBytes")
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *TransactionContext) TransactionEssenceBytes() []byte {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txEssenceBytes")
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *TransactionContext) TransactionID() []byte {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txID")
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *TransactionContext) ConsumeOutput(out *Output, oid OutputID) byte {
	outIdx := v.tree.PushData(out.Bytes(), Path(ConsumedContextBranch, ConsumedContextOutputsBranch))
	idIdx := v.tree.PushData(oid[:], Path(TransactionBranch, TxInputIDsBranch))
	easyutxo.Assert(outIdx == idIdx, "ConsumeOutput: outIdx == idIdx")
	return byte(outIdx)
}

func (v *TransactionContext) ProduceOutput(out *Output) byte {
	return byte(v.tree.PushData(out.Bytes(), Path(TransactionBranch, TxOutputBranch)))
}

func (v *TransactionContext) AddTransactionTimestamp(ts uint32) {
	var d [4]byte
	binary.BigEndian.PutUint32(d[:], ts)
	v.tree.PutDataAtIdx(TxTimestamp, d[:], Path(TransactionBranch))
}

func (v *TransactionContext) UnlockED25519Inputs(pairs []*keyPair) {
	_ = prepareKeyPairs(pairs)
	// TODO
}

type keyPair struct {
	pubKey  ed255192.PublicKey
	privKey ed255192.PrivateKey
}

func prepareKeyPairs(keyPairs []*keyPair) map[string]*keyPair {
	ret := make(map[string]*keyPair)
	for _, kp := range keyPairs {
		addr := AddressDataFromED25519PubKey(kp.pubKey)
		ret[string(addr)] = kp
	}
	return ret
}

func (v *TransactionContext) ForEachOutput(branch lazyslice.TreePath, fun func(out *Output, path lazyslice.TreePath) bool) {
	outputPath := Path(branch, byte(0))
	v.tree.ForEach(func(idx byte, outputData []byte) bool {
		outputPath[2] = idx
		return fun(OutputFromBytes(outputData), outputPath)
	}, branch)
}
