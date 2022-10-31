package state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/indexer"
	"github.com/lunfardo314/easyutxo/ledger/library"
	"golang.org/x/crypto/blake2b"
)

func (v *ValidationContext) evalContext(path []byte) easyfl.GlobalData {
	v.dataContext.SetPath(path)
	switch v.traceOption {
	case TraceOptionNone:
		return easyfl.NewGlobalDataNoTrace(v.dataContext)
	case TraceOptionAll:
		return easyfl.NewGlobalDataTracePrint(v.dataContext)
	case TraceOptionFailedConstraints:
		return easyfl.NewGlobalDataLog(v.dataContext)
	default:
		panic("wrong trace option")
	}
}

// checkConstraint checks the constraint at path. In-line and unlock scripts are ignored
// for 'produces output' context
func (v *ValidationContext) checkConstraint(constraintData []byte, constraintPath lazyslice.TreePath) ([]byte, string, error) {
	var ret []byte
	var name string
	err := common.CatchPanicOrError(func() error {
		var err1 error
		ret, name, err1 = v.evalConstraint(constraintData, constraintPath)
		return err1
	})
	if err != nil {
		return nil, name, err
	}
	return ret, name, nil
}

func (v *ValidationContext) Validate() ([]*indexer.IndexEntry, error) {
	var inSum, outSum uint64
	var err error
	ret := make([]*indexer.IndexEntry, 0)

	err = easyfl.CatchPanicOrError(func() error {
		var err1 error
		inSum, err1 = v.validateConsumedOutputs(&ret)
		return err1
	})
	if err != nil {
		return nil, err
	}
	err = easyfl.CatchPanicOrError(func() error {
		var err1 error
		outSum, err1 = v.validateProducedOutputs(&ret)
		return err1
	})
	if err != nil {
		return nil, err
	}
	err = easyfl.CatchPanicOrError(func() error {
		return v.validateInputCommitment()
	})
	if err != nil {
		return nil, err
	}
	if inSum != outSum {
		return nil, fmt.Errorf("unbalanced amount between inputs and outputs")
	}
	return ret, nil
}

func (v *ValidationContext) validateProducedOutputs(indexRecords *[]*indexer.IndexEntry) (uint64, error) {
	return v.validateOutputs(false, indexRecords)
}

func (v *ValidationContext) validateConsumedOutputs(indexRecords *[]*indexer.IndexEntry) (uint64, error) {
	return v.validateOutputs(true, indexRecords)
}

func (v *ValidationContext) validateOutputs(consumedBranch bool, indexRecords *[]*indexer.IndexEntry) (uint64, error) {
	indexRecs := *indexRecords
	var branch lazyslice.TreePath
	if consumedBranch {
		branch = Path(ConsumedContextBranch, ConsumedContextOutputsBranch)
	} else {
		branch = Path(TransactionBranch, TxOutputBranch)
	}
	var err error
	var sum uint64
	var extraDepositWeight uint32
	path := easyfl.Concat(branch, 0)
	v.tree.ForEach(func(i byte, data []byte) bool {
		path[len(path)-1] = i
		arr := lazyslice.ArrayFromBytes(data, 256)
		if extraDepositWeight, err = v.runOutput(arr, path); err != nil {
			return false
		}
		minDeposit := library.MinimumStorageDeposit(uint32(len(data)), extraDepositWeight)
		var am library.Amount
		am, err = library.AmountFromBytes(arr.At(int(ledger.OutputBlockAmount)))
		if err != nil {
			return false
		}
		amount := am.Amount()
		if amount < minDeposit {
			err = fmt.Errorf("not enough storage deposit in output %s. Minimum %d, got %d",
				PathToString(path), minDeposit, amount)
			return false
		}
		if amount > math.MaxUint64-sum {
			err = fmt.Errorf("validateOutputs @ path %s: uint64 arithmetic overflow", PathToString(path))
			return false
		}
		sum += amount

		// create update command for indexer
		var lock library.Lock
		lock, err = library.LockFromBytes(arr.At(int(ledger.OutputBlockLock)))
		if err != nil {
			return false
		}
		for _, addr := range lock.IndexableTags() {
			indexEntry := &indexer.IndexEntry{
				AccountID: addr.AccountID(),
				Delete:    consumedBranch,
			}
			if consumedBranch {
				indexEntry.OutputID = v.InputID(i)
			} else {
				indexEntry.OutputID = ledger.NewOutputID(v.TransactionID(), i)
			}
			indexRecs = append(indexRecs, indexEntry)
		}
		return true
	}, branch)
	if err != nil {
		return 0, err
	}
	*indexRecords = indexRecs
	return sum, nil
}

func (v *ValidationContext) InputID(idx byte) ledger.OutputID {
	data := v.tree.BytesAtPath(Path(TransactionBranch, TxInputIDsBranch, idx))
	ret, err := ledger.OutputIDFromBytes(data)
	easyfl.AssertNoError(err)
	return ret
}

// runOutput checks constraints of the output one-by-one
func (v *ValidationContext) runOutput(outputArray *lazyslice.Array, path lazyslice.TreePath) (uint32, error) {
	blockPath := easyfl.Concat(path, byte(0))
	var err error
	extraStorageDepositWeight := uint32(0)
	outputArray.ForEach(func(idx int, data []byte) bool {
		blockPath[len(blockPath)-1] = byte(idx)
		var res []byte
		var name string

		res, name, err = v.checkConstraint(data, blockPath)
		if err != nil {
			err = fmt.Errorf("constraint '%s' failed with error '%v'. Path: %s",
				name, err, PathToString(blockPath))
			return false
		}
		if len(res) == 0 {
			err = fmt.Errorf("constraint '%s' failed. Path: %s",
				name, PathToString(blockPath))
			return false
		}
		if len(res) == 4 {
			// 4 bytes long slice returned by the constraint is interpreted as 'true' and as uint32 extraStorageWeight
			extraStorageDepositWeight += binary.BigEndian.Uint32(res)
		}
		return true
	})
	if err != nil {
		return 0, err
	}
	return extraStorageDepositWeight, nil
}

func (v *ValidationContext) validateInputCommitment() error {
	consumedOutputBytes := v.tree.BytesAtPath(Path(ConsumedContextBranch, ConsumedContextOutputsBranch))
	h := blake2b.Sum256(consumedOutputBytes)
	inputCommitment := v.tree.BytesAtPath(Path(TransactionBranch, TxInputCommitment))
	if !bytes.Equal(h[:], inputCommitment) {
		return fmt.Errorf("consumed input hash %v not equal to input commitment %v",
			easyfl.Fmt(h[:]), easyfl.Fmt(inputCommitment))
	}
	return nil
}

func PathToString(path []byte) string {
	ret := "@"
	if len(path) == 0 {
		return ret + ".nil"
	}
	if len(path) >= 1 {
		switch path[0] {
		case TransactionBranch:
			ret += ".tx"
			if len(path) >= 2 {
				switch path[1] {
				case TxUnlockParamsBranch:
					ret += ".unlock"
				case TxInputIDsBranch:
					ret += ".inID"
				case TxOutputBranch:
					ret += ".out"
				case TxTimestamp:
					ret += ".ts"
				case TxInputCommitment:
					ret += ".inhash"
				default:
					ret += "WRONG[1]"
				}
			}
			if len(path) >= 3 {
				ret += fmt.Sprintf("[%d]", path[2])
			}
			if len(path) >= 4 {
				ret += fmt.Sprintf(".block[%d]", path[3])
			}
			if len(path) >= 5 {
				ret += fmt.Sprintf("..%v", path[4:])
			}
		case ConsumedContextBranch:
			ret += ".consumed"
			if len(path) >= 2 {
				if path[1] != 0 {
					ret += ".WRONG[1]"
				} else {
					ret += ".[0]"
				}
			}
			if len(path) >= 3 {
				ret += fmt.Sprintf(".out[%d]", path[2])
			}
			if len(path) >= 4 {
				ret += fmt.Sprintf(".block[%d]", path[3])
			}
			if len(path) >= 5 {
				ret += fmt.Sprintf("..%v", path[4:])
			}
		default:
			ret += ".WRONG[0]"
		}
	}
	return ret
}

func constraintName(binCode []byte) string {
	if binCode[0] == 0 {
		return "array_constraint"
	}
	prefix, err := easyfl.ParseCallPrefixFromBinary(binCode)
	if err != nil {
		return fmt.Sprintf("unknown_constraint(%s)", easyfl.Fmt(binCode))
	}
	name, found := library.NameByPrefix(prefix)
	if found {
		return name
	}
	return fmt.Sprintf("constraint_call_prefix(%s)", easyfl.Fmt(prefix))
}

func (v *ValidationContext) evalConstraint(constr []byte, path lazyslice.TreePath) ([]byte, string, error) {
	if len(constr) == 0 {
		return nil, "", fmt.Errorf("constraint can't be empty")
	}
	var err error
	name := constraintName(constr)
	ctx := v.evalContext(path)
	if ctx.Trace() {
		ctx.PutTrace(fmt.Sprintf("--- check constraint '%s' at path %s", name, PathToString(path)))
	}

	var ret []byte

	if constr[0] != 0 {
		// inline constraint. Binary code cannot start with 0-byte
		ret, err = easyfl.EvalFromBinary(ctx, constr)
	} else {
		// array constraint
		arr := lazyslice.ArrayFromBytes(constr[1:], 256)
		if arr.NumElements() == 0 {
			err = fmt.Errorf("can't evaluate empty array")
		} else {
			binCode := arr.At(0)
			args := make([][]byte, arr.NumElements()-1)
			for i := 1; i < arr.NumElements(); i++ {
				args[i] = arr.At(i)
			}
			ret, err = easyfl.EvalFromBinary(ctx, binCode, args...)
		}
	}

	if ctx.Trace() {
		if err != nil {
			ctx.PutTrace(fmt.Sprintf("--- constraint '%s' at path %s: FAILED with '%v'", name, PathToString(path), err))
			ctx.(*easyfl.GlobalDataLog).PrintLog()
		} else {
			if len(ret) == 0 {
				ctx.PutTrace(fmt.Sprintf("--- constraint '%s' at path %s: FAILED", name, PathToString(path)))
				ctx.(*easyfl.GlobalDataLog).PrintLog()
			} else {
				ctx.PutTrace(fmt.Sprintf("--- constraint '%s' at path %s: OK", name, PathToString(path)))
			}
		}
	}

	return ret, name, err
}
