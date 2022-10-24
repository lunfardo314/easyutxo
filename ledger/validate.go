package ledger

import (
	"bytes"
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

func (v *ValidationContext) Eval(source string, path []byte) ([]byte, error) {
	return easyfl.EvalFromSource(v.dataContext(path), source)
}

func (v *ValidationContext) MustEval(source string, path []byte) []byte {
	ret, err := v.Eval(source, path)
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *ValidationContext) CheckConstraint(source string, path []byte) error {
	return easyfl.CatchPanicOrError(func() error {
		res := v.MustEval(source, path)
		if len(res) == 0 {
			return fmt.Errorf("constraint '%s' failed", source)
		}
		return nil
	})
}

// Invoke checks the constraint at path. In-line and unlock scripts are ignored
// for 'produces output' context
func (v *ValidationContext) Invoke(invocationPath lazyslice.TreePath) ([]byte, string, error) {
	binScript, name, runYN := v.parseInvocationScript(invocationPath)
	if !runYN {
		// inline and unlock scripts are ignored in 'produced output' context
		return nil, name, nil
	}
	// it is either consumed output context or it is an embedded constraint
	f, err := easyfl.ExpressionFromBinary(binScript)
	if err != nil {
		panic(err)
	}
	ctx := v.dataContext(invocationPath)
	if ctx.Trace() {
		ctx.PutTrace(fmt.Sprintf("--- check constraint '%s' at path %s", name, PathToString(invocationPath)))
	}
	var ret []byte
	err = common.CatchPanicOrError(func() error {
		ret = easyfl.EvalExpression(ctx, f)
		return nil
	})
	if ctx.Trace() {
		if err != nil {
			ctx.PutTrace(fmt.Sprintf("--- constraint '%s' %s: FAIL with '%v'",
				name, PathToString(invocationPath), err))
			ctx.(*easyfl.GlobalDataLog).PrintLog()
		} else {
			if len(ret) == 0 {
				ctx.PutTrace(fmt.Sprintf("--- constraint '%s' %s: FAIL", name, PathToString(invocationPath)))
				ctx.(*easyfl.GlobalDataLog).PrintLog()
			} else {
				ctx.PutTrace(fmt.Sprintf("--- constraint '%s' %s: OK", name, PathToString(invocationPath)))
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
	return v.validateOutputs(Path(TransactionBranch, TxOutputBranch))
}

func (v *ValidationContext) validateConsumedOutputs() (uint64, error) {
	return v.validateOutputs(Path(ConsumedContextBranch, ConsumedContextOutputsBranch))
}

func (v *ValidationContext) validateOutputs(branch lazyslice.TreePath) (uint64, error) {
	var err error
	var sum uint64
	v.ForEachOutput(branch, func(out *Output, path lazyslice.TreePath) bool {
		if err = v.runOutput(out, path); err != nil {
			return false
		}
		if out.Amount > math.MaxUint64-sum {
			err = fmt.Errorf("validateOutputs @ branch %v: uint64 arithmetic overflow", branch)
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

func (v *ValidationContext) runOutput(out *Output, path lazyslice.TreePath) error {
	mainBlockBytes := out.Constraint(OutputBlockMain)
	if len(mainBlockBytes) != mainConstraintSize || ConstraintType(mainBlockBytes[0]) != ConstraintTypeMain {
		// we enforce presence of the main constraint, the rest is checked by it
		return fmt.Errorf("wrong main constraint")
	}
	blockPath := easyfl.Concat(path, byte(0))
	var err error
	out.ForEachConstraint(func(idx byte, constraint Constraint) bool {
		blockPath[len(blockPath)-1] = idx
		var res []byte
		var name string

		res, name, err = v.Invoke(blockPath)
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
		return true
	})
	return err
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
