package ledger

import (
	ed255192 "crypto/ed25519"
	"fmt"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger/globalpath"
)

type GlobalContext struct {
	tree *lazyslice.Tree
}

const (
	InvocationTypeInline = byte(iota)
	InvocationTypeLocalLib
	InvocationTypeFirstGlobal
)

func invokeConstraint(tree *lazyslice.Tree, path lazyslice.TreePath) []byte {
	return GlobalContextFromTree(tree).Invoke(path)
}

func (v *GlobalContext) Tree() *lazyslice.Tree {
	return v.tree
}

func (v *GlobalContext) Transaction() *Transaction {
	return FromBytes(v.tree.BytesAtPath(globalpath.Transaction))
}

func (v *GlobalContext) Output(outputGroup byte, idx byte) *Output {
	return &Output{
		tree: lazyslice.TreeFromBytes(v.tree.BytesAtPath(globalpath.TransactionOutput(outputGroup, idx))),
	}
}

func (v *GlobalContext) ConsumedOutput(idx byte) *Output {
	return &Output{
		tree: lazyslice.TreeFromBytes(v.tree.GetDataAtIdx(idx, globalpath.ConsumedOutputs)),
	}
}

func (v *GlobalContext) CodeFromGlobalLibrary(idx byte) []byte {
	return v.tree.GetDataAtIdx(idx, globalpath.ConsumedLibrary)
}

func (v *GlobalContext) CodeFromLocalLibrary(idx byte) []byte {
	return v.Transaction().CodeFromLocalLibrary(idx)
}

func (v *GlobalContext) parseInvocationCode(invocationFullPath lazyslice.TreePath) []byte {
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

// GlobalContextFromTransaction finds all inputs in the ledger state.
// Creates a tree with ledger at long index 0 and all inputs at long index 1
//
func GlobalContextFromTransaction(txBytes []byte, ledgerState LedgerState) (*GlobalContext, error) {
	tx := FromBytes(txBytes)
	ret := &GlobalContext{tree: lazyslice.TreeEmpty()}
	ret.tree.PushEmptySubtrees(2, nil)
	ret.tree.PutSubtreeAtIdx(tx.Tree(), globalpath.TransactionIndex, nil)                                 // #0 transaction
	ret.tree.PushEmptySubtrees(2, globalpath.Consumed)                                                    // #1 consumed context (inputs, library)
	ret.tree.PutSubtreeAtIdx(lazyslice.TreeEmpty(), globalpath.ConsumedOutputsIndex, globalpath.Consumed) // 1 @ 0 consumed outputs
	ret.tree.PutSubtreeAtIdx(constraintTree, globalpath.ConsumedLibraryIndex, globalpath.Consumed)        // 1 @ 1 script library tree

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

func GlobalContextFromTree(dataTree *lazyslice.Tree) *GlobalContext {
	return &GlobalContext{
		tree: dataTree,
	}
}

func NewGlobalContext() *GlobalContext {
	tx := NewTransaction()
	ret, err := GlobalContextFromTransaction(tx.Bytes(), nil)
	easyutxo.AssertNoError(err)
	return ret
}

func (v *GlobalContext) rootContext() *DataContext {
	return NewDataContext(v.tree, nil)
}

func (v *GlobalContext) TransactionBytes() []byte {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txBytes")
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *GlobalContext) TransactionEssenceBytes() []byte {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txEssenceBytes")
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *GlobalContext) TransactionID() []byte {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txid")
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *GlobalContext) Validate() error {
	return nil
}

func (v *GlobalContext) Eval(source string, path []byte) ([]byte, error) {
	return easyfl.EvalFromSource(NewDataContext(v.tree, path), source)
}

func (v *GlobalContext) MustEval(source string, path []byte) []byte {
	ret, err := v.Eval(source, path)
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *GlobalContext) Invoke(invocationPath lazyslice.TreePath) []byte {
	code := v.parseInvocationCode(v.tree.BytesAtPath(invocationPath))
	f, err := easyfl.ExpressionFromBinary(code)
	if err != nil {
		panic(err)
	}
	ctx := NewDataContext(v.tree, invocationPath)
	return easyfl.EvalExpression(ctx, f)
}

func (v *GlobalContext) ConsumeOutput(out *Output, oid OutputID) byte {
	outIdx := v.tree.PushData(out.Bytes(), globalpath.ConsumedOutputs)
	idIdx := v.tree.PushData(oid[:], globalpath.TxInputIDs)
	easyutxo.Assert(outIdx == idIdx, "ConsumeOutput: outIdx == idIdx")
	return byte(outIdx)
}

func (v *GlobalContext) ProduceOutput(out *Output, outputGroup byte) byte {
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

func (v *GlobalContext) UnlockED25519Inputs(pairs []*keyPair) {
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
