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
	GlobalInputsLong       = lazyslice.Path(ValidationCtxInputsIndex)
	GlobalTransaction      = lazyslice.Path(ValidationCtxTransactionIndex)
	GlobalOutputGroups     = lazyslice.Path(ValidationCtxTransactionIndex, TxTreeIndexOutputGroups)
	GlobalInputIDsLong     = lazyslice.Path(ValidationCtxTransactionIndex, TxTreeIndexInputIDsLong)
	GlobalUnlockParamsLong = lazyslice.Path(ValidationCtxTransactionIndex, TxTreeIndexUnlockParamsLong)
)

func GlobalOutput(outputGroup, idx byte) lazyslice.TreePath {
	return lazyslice.PathAppend(GlobalOutputGroups, outputGroup, idx)
}

func GlobalInput(idx uint16) lazyslice.TreePath {
	return lazyslice.PathAppend(GlobalInputsLong, easyutxo.EncodeInteger(idx)...)
}

func IsGlobalInputPath(path []byte) bool {
	return path[0] == ValidationCtxInputsIndex
}

func IsGlobalOutputPath(path []byte) bool {
	return path[0] == ValidationCtxTransactionIndex
}
