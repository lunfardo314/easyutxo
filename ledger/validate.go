package ledger

import (
	"bytes"
	"fmt"
	"math"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"golang.org/x/crypto/blake2b"
)

func (v *TransactionContext) Validate() error {
	return easyutxo.CatchPanicOrError(func() error {
		if len(v.MustEval("equal(len8(txTimestampBytes),4)", nil)) == 0 {
			return fmt.Errorf("wrong transaction timestamp")
		}
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

func checkMandatoryConstraints(out *Output) error {
	amountBlockBytes := out.BlockBytes(OutputBlockTokens)
	if len(amountBlockBytes) != 9 ||
		amountBlockBytes[0] != ConstraintTokens ||
		easyutxo.All0(amountBlockBytes[1:]) {
		return fmt.Errorf("wrong amount")
	}
	timestampBlockBytes := out.BlockBytes(OutputBlockTimestamp)
	if len(timestampBlockBytes) != 5 ||
		timestampBlockBytes[0] != ConstraintTimestamp {
		return fmt.Errorf("wrong timestamp")
	}
	return nil
}

func (v *TransactionContext) validateProducedOutputs() (uint64, error) {
	var err error
	var sum uint64
	v.ForEachProducedOutput(func(out *Output, path lazyslice.TreePath) bool {
		if err = checkMandatoryConstraints(out); err != nil {
			return false
		}
		a := out.Amount()
		if a > math.MaxUint64-sum {
			err = fmt.Errorf("validateProducedOutputs: uint64 arithmetic overflow")
			return false
		}
		sum += out.Amount()
		// TODO invoke all block constraints
		return false
	})
	if err != nil {
		return 0, err
	}
	return sum, nil
}

func (v *TransactionContext) validateConsumedOutputs() (uint64, error) {
	var err error
	var sum uint64
	v.ForEachConsumedOutput(func(out *Output, path lazyslice.TreePath) bool {
		if err = checkMandatoryConstraints(out); err != nil {
			return false
		}
		a := out.Amount()
		if a > math.MaxUint64-sum {
			err = fmt.Errorf("validateConsumedOutputs: uint64 arithmetic overflow")
			return false
		}
		sum += out.Amount()
		// TODO invoke all block constraints
		return false
	})
	if err != nil {
		return 0, err
	}
	return sum, nil
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
