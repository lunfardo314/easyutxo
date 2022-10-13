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

func (v *TransactionContext) Eval(source string, path []byte) ([]byte, error) {
	return easyfl.EvalFromSource(NewDataContext(v.tree, path), source)
}

func (v *TransactionContext) MustEval(source string, path []byte) []byte {
	ret, err := v.Eval(source, path)
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *TransactionContext) CheckConstraint(source string, path []byte) error {
	return easyutxo.CatchPanicOrError(func() error {
		res := v.MustEval(source, path)
		if len(res) == 0 {
			return fmt.Errorf("constraint '%s' failed", source)
		}
		return nil
	})
}

func (v *TransactionContext) Invoke(invocationPath lazyslice.TreePath) []byte {
	code := v.parseInvocationCode(v.tree.BytesAtPath(invocationPath))
	f, err := easyfl.ExpressionFromBinary(code)
	if err != nil {
		panic(err)
	}
	ctx := NewDataContext(v.tree, invocationPath)
	return easyfl.EvalExpression(ctx, f)
}

func (v *TransactionContext) Validate() error {
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

func (v *TransactionContext) validateProducedOutputs() (uint64, error) {
	return v.validateOutputs(Path(TransactionBranch, TxOutputBranch))
}

func (v *TransactionContext) validateConsumedOutputs() (uint64, error) {
	return v.validateOutputs(Path(ConsumedContextBranch, ConsumedContextOutputsBranch))
}

func (v *TransactionContext) validateOutputs(branch lazyslice.TreePath) (uint64, error) {
	var err error
	var sum uint64
	v.ForEachOutput(branch, func(out *Output, path lazyslice.TreePath) bool {
		if err = v.runOutput(out, path); err != nil {
			return false
		}
		a := out.Amount()
		if a > math.MaxUint64-sum {
			err = fmt.Errorf("validateOutputs @ branch %v: uint64 arithmetic overflow", branch)
			return false
		}
		sum += out.Amount()
		return true
	})
	if err != nil {
		return 0, err
	}
	return sum, nil
}

func (v *TransactionContext) runOutput(out *Output, path lazyslice.TreePath) error {
	mainBlockBytes := out.BlockBytes(OutputBlockMain)
	if len(mainBlockBytes) != 1+4+8 || mainBlockBytes[0] != 0 {
		return fmt.Errorf("wrong main constraint")
	}
	if len(out.BlockBytes(OutputBlockAddress)) < 1 {
		return fmt.Errorf("wrong address constraint")
	}
	blockPath := easyutxo.Concat(path, byte(0))
	var err error
	out.ForEachBlock(func(idx byte, blockData []byte) bool {
		blockPath[len(blockPath)-1] = idx
		res := v.Invoke(blockPath)
		if len(res) == 0 {
			err = fmt.Errorf("constraint @ path %v failed", blockPath)
			return false
		}
		return true
	})
	return err
}

func (v *TransactionContext) validateInputCommitment() error {
	consumedOutputBytes := v.Tree().BytesAtPath(Path(ConsumedContextBranch, ConsumedContextOutputsBranch))
	h := blake2b.Sum256(consumedOutputBytes)
	inputCommitment := v.Tree().BytesAtPath(Path(TransactionBranch, TxInputCommitment))
	if !bytes.Equal(h[:], inputCommitment) {
		return fmt.Errorf("consumed input hash not equal to input commitment")
	}
	return nil
}
