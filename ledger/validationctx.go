package ledger

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger/globalpath"
	"github.com/lunfardo314/easyutxo/ledger/library"
)

type GlobalContext struct {
	tree *lazyslice.Tree
}

const (
	InvocationTypeInline = byte(iota)
	InvocationTypeLocalLib
	InvocationTypeFirstGlobal
)

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

func (v *GlobalContext) parseInvocation(invocationFullPath lazyslice.TreePath) ([]byte, []byte) {
	invocation := v.tree.BytesAtPath(invocationFullPath)
	if len(invocation) < 1 {
		panic("empty invocation")
	}
	switch invocation[0] {
	case InvocationTypeLocalLib:
		if len(invocation) < 2 {
			panic("wrong invocation")
		}
		return v.CodeFromLocalLibrary(invocation[1]), invocation[2:]
	case InvocationTypeInline:
		return invocation[1:], nil
	}
	return v.CodeFromGlobalLibrary(invocation[0]), invocation[1:]
}

func (v *GlobalContext) Invoke(invocationPath lazyslice.TreePath) []byte {
	code, data := v.parseInvocation(v.tree.BytesAtPath(invocationPath))
	ctx := library.NewRunContext(v.tree, invocationPath, data)
	f, err := easyfl.FormulaTreeFromBinary(library.Library, code)
	if err != nil {
		panic(err)
	}
	return ctx.Eval(f)
}

// CreateGlobalContext finds all inputs in the ledger state.
// Creates a tree with ledger at long index 0 and all inputs at long index 1
//
func CreateGlobalContext(tx *Transaction, ledgerState LedgerState) (*GlobalContext, error) {

	ret := &GlobalContext{tree: lazyslice.TreeEmpty()}
	ret.tree.PushEmptySubtrees(2, nil)
	ret.tree.PutSubtreeAtIdx(tx.Tree(), globalpath.TransactionIndex, nil)                                     // #0 transaction
	ret.tree.PutSubtreeAtIdx(lazyslice.TreeEmpty(), globalpath.ConsumedIndex, nil)                            // #1 consumed context (inputs, library)
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
	return ret, nil
}
