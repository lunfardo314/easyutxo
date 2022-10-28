package ledger

import (
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

// ValidationContext is a data structure, which contains transaction, consumed outputs and constraint library
type ValidationContext struct {
	tree        *lazyslice.Tree
	traceOption int
}

var Path = lazyslice.Path

// Top level branches
const (
	TransactionBranch = byte(iota)
	ConsumedContextBranch
)

const (
	ConsumedContextOutputsBranch = byte(iota)
)

const (
	TraceOptionNone = iota
	TraceOptionAll
	TraceOptionFailedConstraints
)

// ValidationContextFromTransaction constructs lazytree from transaction bytes and consumed outputs
func ValidationContextFromTransaction(txBytes []byte, ledgerState StateAccess, traceOption ...int) (*ValidationContext, error) {
	txBranch := lazyslice.ArrayFromBytes(txBytes, int(TxTreeIndexMax))
	inputIDs := lazyslice.ArrayFromBytes(txBranch.At(int(TxInputIDsBranch)), 256)

	var err error
	var oid OutputID

	consumedOutputsArray := lazyslice.EmptyArray(256)
	ids := make(map[string]struct{})
	inputIDs.ForEach(func(i int, data []byte) bool {
		if oid, err = OutputIDFromBytes(data); err != nil {
			return false
		}
		// check repeating inputIDs
		if _, repeating := ids[string(data)]; repeating {
			err = fmt.Errorf("repeating input ID: %s", oid.String())
			return false
		}
		ids[string(data)] = struct{}{}

		od, ok := ledgerState.GetUTXO(&oid)
		if !ok {
			err = fmt.Errorf("input not found: %s", oid.String())
			return false
		}
		consumedOutputsArray.Push(od)
		return true
	})
	if err != nil {
		return nil, err
	}
	ctx := lazyslice.MakeArray(
		txBytes, // TransactionBranch = 0
		lazyslice.MakeArray(consumedOutputsArray), // ConsumedContextBranch = 1
	)
	ret := &ValidationContext{
		tree:        ctx.AsTree(),
		traceOption: TraceOptionNone,
	}
	if len(traceOption) > 0 {
		ret.traceOption = traceOption[0]
	}
	return ret, nil
}

const (
	unlockScriptName = "(unlock script constraint)"
	inlineScriptName = "(in-line script constraint)"
)

// unlockScriptBinary finds script from unlock block
func (v *ValidationContext) unlockScriptBinary(invocationFullPath lazyslice.TreePath) []byte {
	unlockBlockPath := easyfl.Concat(invocationFullPath)
	unlockBlockPath[1] = TxUnlockParamsBranch
	return v.tree.BytesAtPath(unlockBlockPath)
}

func (v *ValidationContext) rootContext() easyfl.GlobalData {
	return v.dataContext(nil)
}

func (v *ValidationContext) TransactionBytes() []byte {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txBytes")
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *ValidationContext) TransactionEssenceBytes() []byte {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txEssenceBytes")
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *ValidationContext) TransactionID() TransactionID {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txID")
	if err != nil {
		panic(err)
	}
	var txid TransactionID
	easyfl.Assert(len(txid[:]) == len(ret), "wrong data length")
	copy(txid[:], ret)
	return txid
}

func (v *ValidationContext) InputCommitment() []byte {
	return v.tree.BytesAtPath(Path(TransactionBranch, TxInputCommitment))
}

func (v *ValidationContext) ConsumedOutput(idx byte) *Output {
	ret, err := OutputFromBytes(v.tree.BytesAtPath(Path(ConsumedContextBranch, ConsumedContextOutputsBranch, idx)))
	easyfl.AssertNoError(err)
	return ret
}

func (v *ValidationContext) ForEachOutput(branch lazyslice.TreePath, fun func(out *Output, byteSize uint32, path lazyslice.TreePath) bool) {
	outputPath := easyfl.Concat(branch, byte(0))
	v.tree.ForEach(func(idx byte, outputData []byte) bool {
		outputPath[2] = idx
		out, err := OutputFromBytes(outputData)
		easyfl.AssertNoError(err)
		return fun(out, uint32(len(outputData)), outputPath)
	}, branch)
}

func (v *ValidationContext) ForEachInputID(fun func(idx byte, oid *OutputID) bool) {
	v.tree.ForEach(func(i byte, data []byte) bool {
		oid, err := OutputIDFromBytes(data)
		easyfl.AssertNoError(err)
		if !fun(i, &oid) {
			return false
		}
		return true
	}, Path(TransactionBranch, TxInputIDsBranch))
}

func (v *ValidationContext) ForEachProducedOutput(fun func(out *Output, byteSize uint32, idx byte) bool) {
	v.ForEachOutput([]byte{TransactionBranch, TxOutputBranch}, func(out *Output, byteSize uint32, path lazyslice.TreePath) bool {
		return !fun(out, byteSize, path[len(path)-1])
	})
}

func (v *ValidationContext) NumProducedOutputs() byte {
	return byte(v.tree.NumElements([]byte{TransactionBranch, TxOutputBranch}))
}
