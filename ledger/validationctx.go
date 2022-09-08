package ledger

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger/globalpath"
	"github.com/lunfardo314/easyutxo/ledger/library"
	"github.com/lunfardo314/easyutxo/ledger/opcodes"
)

type GlobalContext struct {
	tree *lazyslice.Tree
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

func (v *GlobalContext) ParseInvocation(invocationFullPath lazyslice.TreePath) (byte, []byte, []byte) {
	invocation := v.tree.BytesAtPath(invocationFullPath)
	switch invocation[0] {
	case library.CodeReservedForLocalInvocations:
		return invocation[1], v.CodeFromLocalLibrary(invocation[1]), invocation[2:]
	case library.CodeReservedForInlineInvocations:
		return invocation[1], invocation[1:], nil
	}
	return invocation[1], v.CodeFromGlobalLibrary(invocation[0]), invocation[1:]
}

func (v *GlobalContext) RunScript(invocationPath lazyslice.TreePath, invocationIndex byte) {
	invocationCode, code, data := v.ParseInvocation(v.tree.BytesAtPath(invocationPath))
	engine.Run(&engine.ScriptInvocationContext{
		Opcodes:            opcodes.All,
		Ctx:                v.Tree(),
		InvocationCode:     invocationCode,
		InvocationFullPath: invocationPath,
		InvocationIdx:      invocationIndex,
		Code:               code,
		Data:               data,
		Trace:              false,
	})
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
	ret.tree.PutSubtreeAtIdx(library.ScriptLibrary, globalpath.ConsumedLibraryIndex, globalpath.Consumed)     // 1 @ 1 script library tree

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
