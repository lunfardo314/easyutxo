package transaction

import (
	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

type ValidationContext struct {
	tree *lazyslice.Tree
}

// indices of ValidationContext parts
const (
	ValidationCtxTransactionIndex = byte(iota)
	ValidationCtxInputsIndex
	ValidationCtxGlobalLibraryIndex
	ValidationCtxIndexMax
)

func (v *ValidationContext) Tree() *lazyslice.Tree {
	return v.tree
}

func (v *ValidationContext) Transaction() *Transaction {
	return FromBytes(v.tree.BytesAtPath(Path(ValidationCtxTransactionIndex)))
}

func (v *ValidationContext) Output(outputContext byte, idx byte) *Output {
	return &Output{
		tree: lazyslice.TreeFromBytes(v.tree.BytesAtPath(Path(ValidationCtxTransactionIndex, TxTreeIndexOutputGroups, outputContext, idx))),
	}
}

func (v *ValidationContext) ConsumedOutput(idx uint16) *Output {
	return &Output{
		tree: lazyslice.TreeFromBytes(v.tree.GetBytesAtIdxLong(idx, Path(ValidationCtxInputsIndex))),
	}
}

func (v *ValidationContext) CodeFromGlobalLibrary(idx byte) []byte {
	return v.tree.GetDataAtIdx(idx, Path(ValidationCtxGlobalLibraryIndex))
}

func (v *ValidationContext) CodeFromLocalLibrary(idx byte) []byte {
	return v.Transaction().CodeFromLocalLibrary(idx)
}

func (v *ValidationContext) ParseInvocation(invocationFullPath lazyslice.TreePath) ([]byte, []byte) {
	invocation := v.tree.BytesAtPath(invocationFullPath)
	switch invocation[0] {
	case LibraryCodeReservedForLocalInvocations:
		return v.CodeFromLocalLibrary(invocation[1]), invocation[1:]
	case LibraryCodeReservedForInlineInvocations:
		return invocation[1:], nil
	}
	return v.CodeFromGlobalLibrary(invocation[0]), invocation[1:]
}

func (v *ValidationContext) Invoke(invocationPath lazyslice.TreePath) {
	code, data := v.ParseInvocation(v.tree.BytesAtPath(invocationPath))
	engine.Run(v.Tree(), invocationPath, code, data)
}
