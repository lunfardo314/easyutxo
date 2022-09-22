package globalpath

import (
	"bytes"
	"encoding/binary"

	"github.com/lunfardo314/easyutxo/lazyslice"
)

// Global tree has 2 branches at root: Transaction and Consumed

const (
	TransactionIndex = byte(iota)
	ConsumedIndex
)

// ValidationInputs tree has 2 branches at root: OutputsLong and Library

const (
	ConsumedOutputsIndexLong = byte(iota)
	ConsumedLibraryIndex
)

// Transaction tree 1st level branch indices

const (
	TxUnlockParamsLongIndex = byte(iota)
	TxInputIDsLongIndex
	TxOutputGroupsIndex
	TxTimestampIndex
	TxValidationInputCommitmentIndex
	TxLocalLibraryIndex
	TxTreeIndexMax
)

var (
	Transaction = lazyslice.Path(TransactionIndex)

	TxUnlockParamsLong        = lazyslice.Path(TransactionIndex, TxUnlockParamsLongIndex)
	TxInputIDsLong            = lazyslice.Path(TransactionIndex, TxInputIDsLongIndex)
	TxOutputGroups            = lazyslice.Path(TransactionIndex, TxOutputGroupsIndex)
	TxTimestamp               = lazyslice.Path(TransactionIndex, TxTimestampIndex)
	TxConsumedInputCommitment = lazyslice.Path(TransactionIndex, TxValidationInputCommitmentIndex)
	TxLocalLibrary            = lazyslice.Path(TransactionIndex, TxLocalLibraryIndex)

	Consumed        = lazyslice.Path(ConsumedIndex)
	ConsumedOutputs = lazyslice.Path(ConsumedIndex, ConsumedOutputsIndexLong)
	ConsumedLibrary = lazyslice.Path(ConsumedIndex, ConsumedLibraryIndex)
)

func TransactionOutput(outputGroup, idx byte) lazyslice.TreePath {
	return lazyslice.PathMakeAppend(TxOutputGroups, outputGroup, idx)
}

func TransactionOutputBlock(outputGroup, idxInGroup, blockIdx byte) lazyslice.TreePath {
	return lazyslice.PathMakeAppend(TxOutputGroups, outputGroup, idxInGroup, blockIdx)
}

func ConsumedOutput(idx uint16) lazyslice.TreePath {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], idx)
	return lazyslice.PathMakeAppend(ConsumedOutputs, b[0], b[1])
}

func ConsumedOutputBlock(outputIdx uint16, blockIdx byte) lazyslice.TreePath {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], outputIdx)
	return lazyslice.PathMakeAppend(ConsumedOutputs, b[0], b[1], blockIdx)
}

func IsConsumedOutputContext(path lazyslice.TreePath) bool {
	return len(path) >= 4 && bytes.Equal(ConsumedOutputs, path[:2])
}

func IsTxOutputContext(path lazyslice.TreePath) bool {
	return len(path) >= 4 && bytes.Equal(TxOutputGroups, path[:2])
}

func UnlockBlockPathFromInputPath(inputPath lazyslice.TreePath) lazyslice.TreePath {
	if !IsConsumedOutputContext(inputPath) {
		panic("not the input invocation context")
	}
	ret := make(lazyslice.TreePath, len(inputPath))
	copy(ret, inputPath)
	ret[0] = TransactionIndex
	return ret
}

func SiblingPath(inputPath lazyslice.TreePath, b0, b1 byte) lazyslice.TreePath {
	ret := make(lazyslice.TreePath, len(inputPath))
	copy(ret, inputPath)
	ret[2] = b0
	ret[3] = b1
	return ret
}
