package path

import (
	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

// ValidationContext tree 1st level branch indices
const (
	ValidationCtxTransactionIndex = byte(iota)
	ValidationCtxInputsIndex
	ValidationCtxGlobalLibraryIndex
	ValidationCtxIndexMax
)

// Transaction tree 1st level branch indices

const (
	TxTreeIndexInputIDsLong = byte(iota)
	TxTreeIndexUnlockParamsLong
	TxTreeIndexOutputGroups
	TxTreeIndexTimestamp
	TxTreeIndexContextCommitment
	TxTreeIndexLocalLibrary
	TxTreeIndexMax
)

var (
	GlobalInputsLong        = lazyslice.Path(ValidationCtxInputsIndex)
	GlobalTransaction       = lazyslice.Path(ValidationCtxTransactionIndex)
	GlobalOutputGroups      = lazyslice.Path(ValidationCtxTransactionIndex, TxTreeIndexOutputGroups)
	GlobalInputIDsLong      = lazyslice.Path(ValidationCtxTransactionIndex, TxTreeIndexInputIDsLong)
	GlobalTimestamp         = lazyslice.Path(ValidationCtxTransactionIndex, TxTreeIndexTimestamp)
	GlobalContextCommitment = lazyslice.Path(ValidationCtxTransactionIndex, TxTreeIndexContextCommitment)
	GlobalUnlockParamsLong  = lazyslice.Path(ValidationCtxTransactionIndex, TxTreeIndexUnlockParamsLong)
)

func GlobalOutput(outputGroup, idx byte) lazyslice.TreePath {
	return lazyslice.PathMakeAppend(GlobalOutputGroups, outputGroup, idx)
}

func GlobalInput(idx uint16) lazyslice.TreePath {
	return lazyslice.PathMakeAppend(GlobalInputsLong, easyutxo.EncodeInteger(idx)...)
}

func IsGlobalInputContext(path lazyslice.TreePath) bool {
	return len(path) >= 3 && path[0] == ValidationCtxInputsIndex
}

func IsGlobalOutputContext(path lazyslice.TreePath) bool {
	return len(path) >= 4 && path[0] == ValidationCtxTransactionIndex && path[1] == TxTreeIndexOutputGroups
}

func UnlockBlockPathFromInputInvocationPath(invocationPath lazyslice.TreePath) lazyslice.TreePath {
	if !IsGlobalInputContext(invocationPath) {
		panic("not the input invocation context")
	}
	return lazyslice.PathMakeAppend(GlobalUnlockParamsLong, invocationPath[1], invocationPath[2])
}
