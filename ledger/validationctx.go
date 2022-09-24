package ledger

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger/globalpath"
	"github.com/lunfardo314/easyutxo/ledger/library"
)

type GlobalContext struct {
	tree        *lazyslice.Tree
	rootContext *library.RunContext
}

const (
	InvocationTypeInline = byte(iota)
	InvocationTypeLocalLib
	InvocationTypeFirstGlobal
)

func init() {
	library.RegisterInvokeConstraintFunc(invokeConstraintFunc)
}

func invokeConstraintFunc(tree *lazyslice.Tree, path lazyslice.TreePath) []byte {
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

func (v *GlobalContext) ConsumedOutput(idx uint16) *Output {
	return &Output{
		tree: lazyslice.TreeFromBytes(v.tree.GetBytesAtIdxLong(idx, globalpath.ConsumedOutputs)),
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
	ret.tree.PutSubtreeAtIdx(tx.Tree(), globalpath.TransactionIndex, nil)                                     // #0 transaction
	ret.tree.PushEmptySubtrees(2, globalpath.Consumed)                                                        // #1 consumed context (inputs, library)
	ret.tree.PutSubtreeAtIdx(lazyslice.TreeEmpty(), globalpath.ConsumedOutputsIndexLong, globalpath.Consumed) // 1 @ 0 consumed outputs
	ret.tree.PutSubtreeAtIdx(constraintTree, globalpath.ConsumedLibraryIndex, globalpath.Consumed)            // 1 @ 1 script library tree

	var err error
	tx.ForEachInputID(func(idx uint16, oid OutputID) bool {
		od, ok := ledgerState.GetUTXO(&oid)
		if !ok {
			err = fmt.Errorf("input not found: %s", oid.String())
			return false
		}
		ret.tree.PushLongAtPath(od, globalpath.ConsumedOutputs)
		return true
	})
	if err != nil {
		return nil, err
	}
	ret.rootContext = library.NewGlobalContext(ret.tree, nil)

	return ret, nil
}

func GlobalContextFromTree(dataTree *lazyslice.Tree) *GlobalContext {
	return &GlobalContext{
		tree:        dataTree,
		rootContext: library.NewGlobalContext(dataTree, nil),
	}
}

var (
	TransactionBytes        = library.MustMakeEvalFunc("txBytes")
	TransactionID           = library.MustMakeEvalFunc("txid")
	TransactionEssenceBytes = library.MustMakeEvalFunc("txEssenceBytes")
)

func (v *GlobalContext) TransactionBytes() []byte {
	return TransactionBytes(v.rootContext)
}

func (v *GlobalContext) TransactionEssenceBytes() []byte {
	return TransactionEssenceBytes(v.rootContext)
}

func (v *GlobalContext) TransactionID() []byte {
	return TransactionID(v.rootContext)
}

func (v *GlobalContext) Validate() error {
	return nil
}

func (v *GlobalContext) RunContext(path []byte) *library.RunContext {
	return library.NewGlobalContext(v.tree, path)
}

func (v *GlobalContext) Eval(formulaSource string, path []byte) []byte {
	fun := library.MustMakeEvalFunc(formulaSource)

	return fun(v.RunContext(path))
}

func (v *GlobalContext) Invoke(invocationPath lazyslice.TreePath) []byte {
	code := v.parseInvocationCode(v.tree.BytesAtPath(invocationPath))
	ctx := library.NewGlobalContext(v.tree, invocationPath)
	f, err := easyfl.FormulaTreeFromBinary(library.Library, code)
	if err != nil {
		panic(err)
	}
	return ctx.Eval(f)
}
