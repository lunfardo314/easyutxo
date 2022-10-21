package ledger

import (
	"bytes"
	"fmt"
	"math"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"golang.org/x/crypto/blake2b"
)

func (v *ValidationContext) Eval(source string, path []byte) ([]byte, error) {
	return easyfl.EvalFromSource(NewDataContext(v.tree, path), source)
}

func (v *ValidationContext) MustEval(source string, path []byte) []byte {
	ret, err := v.Eval(source, path)
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *ValidationContext) CheckConstraint(source string, path []byte) error {
	return easyutxo.CatchPanicOrError(func() error {
		res := v.MustEval(source, path)
		if len(res) == 0 {
			return fmt.Errorf("constraint '%s' failed", source)
		}
		return nil
	})
}

func (v *ValidationContext) Invoke(invocationPath lazyslice.TreePath) []byte {
	code, name := v.parseInvocationCode(invocationPath)
	f, err := easyfl.ExpressionFromBinary(code)
	if err != nil {
		panic(err)
	}
	ctx := NewDataContext(v.tree, invocationPath, v.trace)
	if ctx.Trace() {
		ctx.PutTrace(fmt.Sprintf("--- check constraint '%s' at path %s", name, PathToString(invocationPath)))
	}
	ret := easyfl.EvalExpression(ctx, f)
	if ctx.Trace() {
		ok := "OK"
		if len(ret) == 0 {
			ok = "FAIL"
		}
		ctx.PutTrace(fmt.Sprintf("--- constraint '%s' %s: %s", name, PathToString(invocationPath), ok))
	}
	return ret
}

func (v *ValidationContext) Validate() error {
	return easyutxo.CatchPanicOrError(func() error {
		inSum, err := v.validateConsumedOutputs()
		if err != nil {
			return err
		}
		outSum, err := v.validateProducedOutputs()
		if err != nil {
			return err
		}
		if inSum != outSum {
			return fmt.Errorf("unbalanced amount between inputs and outputs")
		}
		if err = v.validateInputCommitment(); err != nil {
			return err
		}
		return nil
	})
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
	if len(mainBlockBytes) != mainConstraintSize || mainBlockBytes[0] != ConstraintMain {
		return fmt.Errorf("wrong main constraint")
	}
	if len(out.Constraint(OutputBlockAddress)) < 1 {
		return fmt.Errorf("wrong address constraint")
	}
	blockPath := easyutxo.Concat(path, byte(0))
	var err error
	out.ForEachConstraint(func(idx byte, constraint Constraint) bool {
		blockPath[len(blockPath)-1] = idx
		res := v.Invoke(blockPath)
		if len(res) == 0 {
			err = fmt.Errorf("constraint failed. Path: %s", PathToString(blockPath))
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
			easyutxo.Hex(h[:]), easyutxo.Hex(inputCommitment))
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
