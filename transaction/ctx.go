package transaction

import (
	"github.com/lunfardo314/easyutxo/lazyslice"
)

type ValidationContext struct {
	tree *lazyslice.Tree
}

const (
	ValidationCtxTxIndex = byte(iota)
	ValidationCtxtInputsIndex
	ValidationCtxScriptLibraryIndex
	ValidationCtxIndexMax
)

func (v *ValidationContext) Tree() *lazyslice.Tree {
	return v.tree
}

func (v *ValidationContext) ConsumedOutput(idx uint16) *Output {
	return &Output{
		tree: lazyslice.TreeFromBytes(v.tree.GetBytesAtIdxLong(idx, Path(ValidationCtxtInputsIndex))),
	}
}

func (v *ValidationContext) Transaction() *Transaction {
	return FromBytes(v.tree.BytesAtPath(Path(ValidationCtxTxIndex)))
}

func (v *ValidationContext) Output(outputContext byte, idx byte) *Output {
	return &Output{
		tree: lazyslice.TreeFromBytes(v.tree.BytesAtPath(Path(ValidationCtxTxIndex, TxTreeIndexOutputsLong, outputContext, idx))),
	}
}
