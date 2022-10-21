package ledger

import (
	"errors"
	"fmt"

	"github.com/lunfardo314/easyfl"
)

type constraintRecord struct {
	name string
	bin  []byte
}

var constraints [256]*constraintRecord

const (
	ConstraintReserved0 = byte(iota)
	ConstraintReserved1
	ConstraintMain
	ConstraintSigLockED25519
)

func init() {
	extendLibrary()

	easyfl.MustExtendMany(SigLockConstraint)
	easyfl.MustExtendMany(MainConstraint)

	mustRegisterConstraint(ConstraintMain, "mainConstraint")
	mustRegisterConstraint(ConstraintSigLockED25519, "sigLocED25519")

	easyfl.PrintLibraryStats()
}

func registerConstraint(invocationCode byte, source string) error {
	if invocationCode <= 1 {
		return errors.New("invocation codes 0 and 1 are reserved")
	}
	if constraints[invocationCode] != nil {
		return fmt.Errorf("repeating invocation code %d: '%s'", invocationCode, source)
	}
	_, numParams, code, err := easyfl.CompileExpression(source)
	if err != nil {
		return err
	}
	if numParams != 0 {
		return fmt.Errorf("formula parameters cannot be used in the constraint: '%s'", source)
	}
	constraints[invocationCode] = &constraintRecord{
		name: source,
		bin:  code,
	}
	fmt.Printf("constraint %d registered: '%s'\n", invocationCode, source)
	return nil
}

func mustRegisterConstraint(invocationCode byte, source string) {
	if err := registerConstraint(invocationCode, source); err != nil {
		panic(err)
	}
}

func mustGetConstraintBinary(idx byte) ([]byte, string) {
	ret := constraints[idx]
	easyfl.Assert(ret != nil, "can't find constraint at index '%d'", idx)
	return ret.bin, ret.name
}
