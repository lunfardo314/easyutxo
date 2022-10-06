package ledger

import (
	ed255192 "crypto/ed25519"
	"fmt"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger/globalpath"
)

// TransactionContext is a data structure, which contains transaction, consumed outputs and constraint library
type TransactionContext struct {
	tree *lazyslice.Tree
}

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
// Creates a tree with ledger at long index 0 and all inputs at long index 1
//
func TransactionContextFromTransaction(txBytes []byte, ledgerState LedgerState) (*TransactionContext, error) {
	tx := FromBytes(txBytes)
	ret := &TransactionContext{tree: lazyslice.TreeEmpty()}
	ret.tree.PushEmptySubtrees(2, nil)
	ret.tree.PutSubtreeAtIdx(tx.Tree(), globalpath.TransactionIndex, nil)                                    // #0 transaction
	ret.tree.PushEmptySubtrees(2, globalpath.Consumed)                                                       // #1 consumed context (inputs, library)
	ret.tree.PutSubtreeAtIdx(lazyslice.TreeEmpty(), globalpath.ConsumedOutputsIndex, globalpath.Consumed)    // 1 @ 0 consumed outputs
	ret.tree.PutSubtreeAtIdx(constraintTree, globalpath.ConsumedConstraintLibraryIndex, globalpath.Consumed) // 1 @ 1 script library tree

	var err error
	tx.ForEachInputID(func(idx byte, oid OutputID) bool {
		od, ok := ledgerState.GetUTXO(&oid)
		if !ok {
			err = fmt.Errorf("input not found: %s", oid.String())
			return false
		}
		ret.tree.PushData(od, globalpath.ConsumedOutputs)
		return true
	})
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
	return FromBytes(v.tree.BytesAtPath(globalpath.Transaction))
}

func (v *TransactionContext) Output(outputGroup byte, idx byte) *Output {
	return &Output{
		tree: lazyslice.TreeFromBytes(v.tree.BytesAtPath(globalpath.TransactionOutput(outputGroup, idx))),
	}
}

func (v *TransactionContext) ConsumedOutput(idx byte) *Output {
	return &Output{
		tree: lazyslice.TreeFromBytes(v.tree.GetDataAtIdx(idx, globalpath.ConsumedOutputs)),
	}
}

func (v *TransactionContext) CodeFromGlobalLibrary(idx byte) []byte {
	return v.tree.GetDataAtIdx(idx, globalpath.ConsumedLibrary)
}

func (v *TransactionContext) CodeFromLocalLibrary(idx byte) []byte {
	return v.Transaction().CodeFromLocalLibrary(idx)
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
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txid")
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *TransactionContext) Eval(source string, path []byte) ([]byte, error) {
	return easyfl.EvalFromSource(NewDataContext(v.tree, path), source)
}

func (v *TransactionContext) MustEval(source string, path []byte) []byte {
	ret, err := v.Eval(source, path)
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *TransactionContext) Invoke(invocationPath lazyslice.TreePath) []byte {
	code := v.parseInvocationCode(v.tree.BytesAtPath(invocationPath))
	f, err := easyfl.ExpressionFromBinary(code)
	if err != nil {
		panic(err)
	}
	ctx := NewDataContext(v.tree, invocationPath)
	return easyfl.EvalExpression(ctx, f)
}

func (v *TransactionContext) ConsumeOutput(out *Output, oid OutputID) byte {
	outIdx := v.tree.PushData(out.Bytes(), globalpath.ConsumedOutputs)
	idIdx := v.tree.PushData(oid[:], globalpath.TxInputIDs)
	easyutxo.Assert(outIdx == idIdx, "ConsumeOutput: outIdx == idIdx")
	return byte(outIdx)
}

func (v *TransactionContext) ProduceOutput(out *Output, outputGroup byte) byte {
	path := easyutxo.Concat(globalpath.TxOutputGroups)
	if v.tree.NumElements(globalpath.TxOutputGroups) > int(outputGroup) {
		// group exists
		path = append(path, outputGroup)
	} else {
		v.tree.PushEmptySubtrees(1, globalpath.TxOutputGroups)
		path = append(path, 0)
	}
	outIdx := v.tree.PushData(out.Bytes(), path)
	return byte(outIdx)
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
		addr := AddressFromED25519PubKey(kp.pubKey)
		ret[string(addr)] = kp
	}
	return ret
}

func (v *TransactionContext) Validate() error {
	if err := v.ValidateProducedOutputs(); err != nil {
		return err
	}
	return nil
}
func (v *TransactionContext) ValidateProducedOutputs() error {
	return nil
}

func (v *TransactionContext) ForEachProducedOutput(fun func(out *Output, path lazyslice.TreePath) bool) {
	outputGroupPath := easyutxo.Concat(globalpath.TxOutputGroups)
	outputGroupPath = append(outputGroupPath, 0)
	outputPath := easyutxo.Concat(outputGroupPath)
	outputPath = append(outputPath, 0)

	v.tree.ForEachIndex(func(outputGroup byte) bool {
		outputGroupPath[len(outputGroupPath)-1] = outputGroup

		v.tree.ForEachSubtree(func(outputIdx byte, subtree *lazyslice.Tree) bool {
			outputPath[len(outputPath)-2] = outputGroup
			outputPath[len(outputPath)-1] = outputIdx
			// subtree is an output subtree
			out := OutputFromTree(subtree)
			return !fun(out, outputPath)
		}, outputGroupPath)
		return true
	}, globalpath.TxOutputGroups)
}
