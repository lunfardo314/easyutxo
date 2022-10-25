package ledger

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"golang.org/x/crypto/blake2b"
)

func (v *ValidationContext) dataContext(path []byte) easyfl.GlobalData {
	switch v.traceOption {
	case TraceOptionNone:
		return easyfl.NewGlobalDataNoTrace(&DataContext{
			dataTree:       v.tree,
			invocationPath: path,
		})
	case TraceOptionAll:
		return easyfl.NewGlobalDataTracePrint(&DataContext{
			dataTree:       v.tree,
			invocationPath: path,
		})
	case TraceOptionFailedConstraints:
		return easyfl.NewGlobalDataLog(&DataContext{
			dataTree:       v.tree,
			invocationPath: path,
		})
	default:
		panic("wrong trace option")
	}
}

// checkConstraint checks the constraint at path. In-line and unlock scripts are ignored
// for 'produces output' context
func (v *ValidationContext) checkConstraint(constraintPath lazyslice.TreePath, isProducedContext bool) ([]byte, string, error) {
	binScript, name, runYN := v.parseConstraintScript(constraintPath, isProducedContext)
	if !runYN {
		// inline and unlock scripts are ignored in 'produced output' context
		return nil, name, nil
	}
	// it is either consumed output context or it is an embedded constraint
	f, err := easyfl.ExpressionFromBinary(binScript)
	if err != nil {
		panic(err)
	}
	ctx := v.dataContext(constraintPath)
	if ctx.Trace() {
		ctx.PutTrace(fmt.Sprintf("--- check constraint '%s' at path %s", name, PathToString(constraintPath)))
	}
	var ret []byte
	err = common.CatchPanicOrError(func() error {
		ret = easyfl.EvalExpression(ctx, f)
		return nil
	})
	if ctx.Trace() {
		if err != nil {
			ctx.PutTrace(fmt.Sprintf("--- constraint '%s' %s: FAIL with '%v'",
				name, PathToString(constraintPath), err))
			ctx.(*easyfl.GlobalDataLog).PrintLog()
		} else {
			if len(ret) == 0 {
				ctx.PutTrace(fmt.Sprintf("--- constraint '%s' %s: FAIL", name, PathToString(constraintPath)))
				ctx.(*easyfl.GlobalDataLog).PrintLog()
			} else {
				ctx.PutTrace(fmt.Sprintf("--- constraint '%s' %s: OK", name, PathToString(constraintPath)))
			}
		}
	}
	return ret, name, err
}

func (v *ValidationContext) Validate() error {
	var inSum, outSum uint64
	var err error
	err = easyfl.CatchPanicOrError(func() error {
		var err1 error
		inSum, err1 = v.validateConsumedOutputs()
		return err1
	})
	if err != nil {
		return err
	}
	err = easyfl.CatchPanicOrError(func() error {
		var err1 error
		outSum, err1 = v.validateProducedOutputs()
		return err1
	})
	if err != nil {
		return err
	}
	err = easyfl.CatchPanicOrError(func() error {
		return v.validateInputCommitment()
	})
	if err != nil {
		return err
	}
	if inSum != outSum {
		return fmt.Errorf("unbalanced amount between inputs and outputs")
	}
	return nil
}

func (v *ValidationContext) validateProducedOutputs() (uint64, error) {
	return v.validateOutputs(Path(TransactionBranch, TxOutputBranch), true)
}

func (v *ValidationContext) validateConsumedOutputs() (uint64, error) {
	return v.validateOutputs(Path(ConsumedContextBranch, ConsumedContextOutputsBranch), false)
}

func (v *ValidationContext) validateOutputs(branch lazyslice.TreePath, isProducedContext bool) (uint64, error) {
	var err error
	var sum uint64
	var extraDepositWeight uint32
	v.ForEachOutput(branch, func(out *Output, byteSize uint32, path lazyslice.TreePath) bool {
		if extraDepositWeight, err = v.runOutput(out, path, isProducedContext); err != nil {
			return false
		}
		minDeposit := MinimumStorageDeposit(byteSize, extraDepositWeight)
		if out.Amount < minDeposit {
			err = fmt.Errorf("not enough storage deposit in output %s. Minimum %d, got %d",
				PathToString(path), minDeposit, out.Amount)
			return false
		}
		if out.Amount > math.MaxUint64-sum {
			err = fmt.Errorf("validateOutputs @ path %s: uint64 arithmetic overflow", PathToString(path))
			return false
		}
		sum += out.Amount
		return true
	})
	if err != nil {
		return 0, err
	}
	return sum, nil
}

// runOutput checks constraints of the output one-by-one
func (v *ValidationContext) runOutput(out *Output, path lazyslice.TreePath, isProducedContext bool) (uint32, error) {
	mainBlockBytes := out.Constraint(OutputBlockMain)
	if len(mainBlockBytes) != mainConstraintSize || ConstraintType(mainBlockBytes[0]) != ConstraintTypeMain {
		// we enforce presence of the main constraint, the rest is checked by it
		return 0, fmt.Errorf("wrong main constraint")
	}
	blockPath := easyfl.Concat(path, byte(0))
	var err error
	extraStorageDepositWeight := uint32(0)

	out.ForEachConstraint(func(idx byte, constraint Constraint) bool {
		blockPath[len(blockPath)-1] = idx
		var res []byte
		var name string

		res, name, err = v.checkConstraint(blockPath, isProducedContext)
		if err != nil {
			err = fmt.Errorf("constraint '%s' failed err='%v'. Path: %s",
				name, err, PathToString(blockPath))
			return false
		}
		if len(res) == 0 {
			err = fmt.Errorf("constraint '%s' failed'. Path: %s",
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
