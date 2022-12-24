package state

import (
	"encoding/binary"
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraints"
	"github.com/lunfardo314/unitrie/common"
	"golang.org/x/crypto/blake2b"
)

// TransactionContext is a data structure, which contains transferable transaction, consumed outputs and constraint library
type TransactionContext struct {
	tree        *lazyslice.Tree
	traceOption int
	// cached values
	dataContext *constraints.DataContext
	txid        ledger.TransactionID
}

var Path = lazyslice.Path

const (
	TraceOptionNone = iota
	TraceOptionAll
	TraceOptionFailedConstraints
)

// TransactionContextFromTransferableBytes constructs lazytree from transaction bytes and consumed outputs
func TransactionContextFromTransferableBytes(txBytes []byte, ledgerState ledger.StateAccess, traceOption ...int) (*TransactionContext, error) {
	txBranch := lazyslice.ArrayFromBytes(txBytes, int(constraints.TxTreeIndexMax))
	inputIDs := lazyslice.ArrayFromBytes(txBranch.At(int(constraints.TxInputIDs)), 256)

	var err error
	var oid ledger.OutputID

	consumedOutputsArray := lazyslice.EmptyArray(256)
	ids := make(map[string]struct{})
	inputIDs.ForEach(func(i int, data []byte) bool {
		if oid, err = ledger.OutputIDFromBytes(data); err != nil {
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
	tree := ctx.AsTree()
	ret := &TransactionContext{
		tree:        tree,
		dataContext: constraints.NewDataContext(tree),
		traceOption: TraceOptionNone,
		txid:        blake2b.Sum256(txBytes),
	}
	if len(traceOption) > 0 {
		ret.traceOption = traceOption[0]
	}
	return ret, nil
}

// unlockScriptBinary finds script from unlock block
func (v *TransactionContext) unlockScriptBinary(invocationFullPath lazyslice.TreePath) []byte {
	unlockBlockPath := common.Concat(invocationFullPath)
	unlockBlockPath[1] = constraints.TxUnlockParams
	return v.tree.BytesAtPath(unlockBlockPath)
}

func (v *TransactionContext) rootContext() easyfl.GlobalData {
	return v.evalContext(nil)
}

func (v *TransactionContext) TransactionBytes() []byte {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txBytes")
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *TransactionContext) TransactionID() ledger.TransactionID {
	return v.txid
}

func (v *TransactionContext) InputCommitment() []byte {
	return v.tree.BytesAtPath(Path(constraints.TransactionBranch, constraints.TxInputCommitment))
}

func (v *TransactionContext) Signature() []byte {
	return v.tree.BytesAtPath(Path(constraints.TransactionBranch, constraints.TxSignature))
}

func (v *TransactionContext) ForEachInputID(fun func(idx byte, oid *ledger.OutputID) bool) {
	v.tree.ForEach(func(i byte, data []byte) bool {
		oid, err := ledger.OutputIDFromBytes(data)
		easyfl.AssertNoError(err)
		if !fun(i, &oid) {
			return false
		}
		return true
	}, Path(constraints.TransactionBranch, constraints.TxInputIDs))
}

func (v *TransactionContext) ConsumedOutputData(idx byte) []byte {
	return v.tree.BytesAtPath(Path(constraints.ConsumedBranch, constraints.ConsumedOutputsBranch, idx))
}

func (v *TransactionContext) UnlockData(idx byte) []byte {
	return v.tree.BytesAtPath(Path(constraints.TransactionBranch, constraints.TxUnlockParams, idx))
}

func (v *TransactionContext) ProducedOutputData(idx byte) []byte {
	return v.tree.BytesAtPath(Path(constraints.TransactionBranch, constraints.TxOutputs, idx))
}

func (v *TransactionContext) NumProducedOutputs() int {
	return v.tree.NumElements([]byte{constraints.TransactionBranch, constraints.TxOutputs})
}

func (v *TransactionContext) NumInputs() int {
	return v.tree.NumElements([]byte{constraints.TransactionBranch, constraints.TxInputIDs})
}

func (v *TransactionContext) InputID(idx byte) ledger.OutputID {
	data := v.tree.BytesAtPath(Path(constraints.TransactionBranch, constraints.TxInputIDs, idx))
	ret, err := ledger.OutputIDFromBytes(data)
	easyfl.AssertNoError(err)
	return ret
}

func (v *TransactionContext) TimestampData() ([]byte, uint32) {
	ret := v.tree.BytesAtPath(Path(constraints.TransactionBranch, constraints.TxTimestamp))
	retTs := uint32(0)
	if len(ret) == 4 {
		retTs = binary.BigEndian.Uint32(ret)
	}
	return ret, retTs
}
