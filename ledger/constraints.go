package ledger

import (
	"errors"
	"fmt"

	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger/library"
)

var constraints [256][]byte
var constraintTree = lazyslice.TreeEmpty()

const (
	ConstraintReserved1 = byte(iota)
	ConstraintReserved2
	ConstraintSigLockED25519
)

func init() {
	library.MustExtendWithMany(SigLockConstraint)

	mustRegisterConstraint(ConstraintSigLockED25519, "sigLocED25519")

	for _, code := range constraints {
		constraintTree.PushData(code, nil)
	}
}

func registerConstraint(invocationCode byte, source string) error {
	if invocationCode <= 1 {
		return errors.New("invocation codes 0 and 1 are reserved")
	}
	if constraints[invocationCode] != nil {
		return fmt.Errorf("repeating invocation code %d: '%s'", invocationCode, source)
	}
	_, numParams, code, err := easyfl.CompileFormula(library.Library, source)
	if err != nil {
		return err
	}
	if numParams != 0 {
		return fmt.Errorf("formula parameters cannot be used in the constraint: '%s'", source)
	}
	constraints[invocationCode] = code
	fmt.Printf("constraint %d registered: '%s'\n", invocationCode, source)
	return nil
}

func mustRegisterConstraint(invocationCode byte, source string) {
	if err := registerConstraint(invocationCode, source); err != nil {
		panic(err)
	}
}
