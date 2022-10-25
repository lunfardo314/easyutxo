package ledger

import (
	"errors"
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

type DataContext struct {
	dataTree       *lazyslice.Tree
	invocationPath lazyslice.TreePath
}

func init() {
	//-------------------------------- standard EasyFL library extensions ------------------------------

	// context access
	easyfl.EmbedShort("@", 0, evalPath, true)
	easyfl.EmbedShort("@Path", 1, evalAtPath, true)
	easyfl.Extend("#vbCost16", "u16/1")

	// LazyArray
	// @Array8 interprets $0 as serialized LazyArray. Takes the $1 element of it. $1 is expected 1-byte
	easyfl.EmbedLong("@Array8", 2, evalAtArray8)

	easyfl.Extend("isConsumedBranch", "equal(slice($0,0,1), 0x0100)")
	easyfl.Extend("isProducedBranch", "equal(slice($0,0,1), 0x0002)")
	easyfl.Extend("consumedOutputPathByIndex", "concat(0x0100,$0)")
	easyfl.Extend("producedOutputPathByIndex", "concat(0x0000,$0)")
	easyfl.Extend("consumedOutputByIndex", "@Path(concat(0x0100,$0))")
	easyfl.Extend("producedOutputByIndex", "@Path(concat(0x0000,$0))")
	easyfl.Extend("consumedLockByOutputIndex", "@Array8(consumedOutputByIndex($0),1)")

	// special transaction related

	easyfl.Extend("txBytes", "@Path(0x00)")
	easyfl.Extend("txID", "blake2b(txBytes)")
	easyfl.Extend("txInputIDsBytes", "@Path(0x0001)")
	easyfl.Extend("txOutputsBytes", "@Path(0x0002)")
	easyfl.Extend("txTimestampBytes", "@Path(0x0003)")
	easyfl.Extend("txInputCommitmentBytes", "@Path(0x0004)")
	easyfl.Extend("txEssenceBytes", "concat(txInputIDsBytes, txOutputsBytes, txInputCommitmentBytes)")
	easyfl.Extend("addrDataED25519FromPubKey", "blake2b($0)")

	easyfl.Extend("selfOutputPath", "slice(@,0,2)")
	easyfl.Extend("selfSiblingBlock", "@Array8(@Path(selfOutputPath), $0)")
	easyfl.Extend("selfOutputBytes", "@Path(selfOutputPath)")

	// unlock param branch (0 - transaction, 0 unlock params)
	easyfl.Extend("unlockParamBranch", "0x0000")
	// invoked output block
	easyfl.Extend("selfConstraint", "@Path(@)")
	// output index of the invocation
	easyfl.Extend("selfOutputIndex", "slice(@, 2, 2)")
	// block index of the invocation
	easyfl.Extend("selfBlockIndex", "tail(@, 3)")
	// branch (2 bytes) of the constraint invocation
	easyfl.Extend("selfBranch", "slice(@,0,1)")
	// output index || block index
	easyfl.Extend("selfConstraintIndex", "slice(@, 2, 3)")
	// invocation output data
	easyfl.Extend("selfConstraintData", "tail(selfConstraint,1)")
	// unlock parameters of the invoked consumed constraint
	easyfl.Extend("selfUnlockParameters", "@Path(concat(unlockParamBranch, selfConstraintIndex))")
	// path referenced by the reference unlock params
	easyfl.Extend("selfReferencedPath", "concat(selfBranch, selfUnlockParameters, selfBlockIndex)")
	// constraint referenced by the referenced path
	easyfl.Extend("selfReferencedConstraint", "@Path(selfReferencedPath)")

	//--------------------------- constraints ------------------------------------

	easyfl.MustExtendMany(MainConstraintSource)
	easyfl.MustExtendMany(SigLockED25519ConstraintSource)
	easyfl.MustExtendMany(SenderConstraintSource)

	mustRegisterConstraint(ConstraintTypeMain, "mainConstraint")
	mustRegisterConstraint(ConstraintTypeSigLockED25519, "sigLockED25519")
	mustRegisterConstraint(ConstraintTypeSender, "senderValid")

	easyfl.PrintLibraryStats()
}

func evalPath(ctx *easyfl.CallParams) []byte {
	return ctx.DataContext().(*DataContext).invocationPath
}

func evalAtPath(ctx *easyfl.CallParams) []byte {
	return ctx.DataContext().(*DataContext).dataTree.BytesAtPath(ctx.Arg(0))
}

func evalAtArray8(ctx *easyfl.CallParams) []byte {
	arr := lazyslice.ArrayFromBytes(ctx.Arg(0))
	idx := ctx.Arg(1)
	if len(idx) != 1 {
		panic("evalAtArray8: 1-byte value expected")
	}
	return arr.At(int(idx[0]))
}

type constraintRecord struct {
	name string
	bin  []byte
}

var constraints = make(map[ConstraintType]*constraintRecord)

type ConstraintType byte

const (
	ConstraintTypeInlineScript = ConstraintType(iota)
	ConstraintTypeUnlockScript
	ConstraintTypeMain
	ConstraintTypeSigLockED25519
	ConstraintTypeSender
)

func registerConstraint(invocationCode ConstraintType, source string) error {
	if invocationCode <= 1 {
		return errors.New("invocation codes 0 and 1 are reserved")
	}
	if _, found := constraints[invocationCode]; found {
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

func mustRegisterConstraint(invocationCode ConstraintType, source string) {
	if err := registerConstraint(invocationCode, source); err != nil {
		panic(err)
	}
}

func mustGetConstraintBinary(idx ConstraintType) ([]byte, string) {
	ret := constraints[idx]
	easyfl.Assert(ret != nil, "can't find constraint at constraintID '%d'", idx)
	return ret.bin, ret.name
}
