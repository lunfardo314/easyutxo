package ledger

import (
	"github.com/iotaledger/trie.go/common"
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
	easyfl.EmbedShort("validAmount", 1, evalValidAmount, true)
	easyfl.Extend("#vbCost16", "u16/1")

	// LazyArray
	// @Array8 interprets $0 as serialized LazyArray with max 256 elements. Takes the $1 element of it. $1 is expected 1-byte
	easyfl.EmbedLong("@Array8", 2, evalAtArray8)

	easyfl.Extend("isConsumedBranch", "equal(slice($0,0,1), 0x0100)")
	easyfl.Extend("isProducedBranch", "equal(slice($0,0,1), 0x0002)")
	easyfl.Extend("consumedOutputPathByIndex", "concat(0x0100,$0)")
	easyfl.Extend("producedOutputPathByIndex", "concat(0x0000,$0)")
	easyfl.Extend("consumedOutputByIndex", "@Path(concat(0x0100,$0))")
	easyfl.Extend("producedOutputByIndex", "@Path(concat(0x0000,$0))")
	easyfl.Extend("consumedLockByOutputIndex", "@Array8(consumedOutputByIndex($0),2)")

	// special transaction related

	easyfl.Extend("txBytes", "@Path(0x00)")
	easyfl.Extend("txID", "blake2b(txBytes)")
	easyfl.Extend("txInputIDsBytes", "@Path(0x0001)")
	easyfl.Extend("txOutputsBytes", "@Path(0x0002)")
	easyfl.Extend("txTimestampBytes", "@Path(0x0003)")
	easyfl.Extend("txInputCommitmentBytes", "@Path(0x0004)")
	easyfl.Extend("txEssenceBytes", "concat(txInputIDsBytes, txOutputsBytes, txInputCommitmentBytes)")

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
	// data of a constraint
	easyfl.Extend("constraintData", "tail($0,1)")
	// invocation output data
	easyfl.Extend("selfConstraintData", "constraintData(selfConstraint)")
	// unlock parameters of the invoked consumed constraint
	easyfl.Extend("selfUnlockParameters", "@Path(concat(unlockParamBranch, selfConstraintIndex))")
	// path referenced by the reference unlock params
	easyfl.Extend("selfReferencedPath", "concat(selfBranch, selfUnlockParameters, selfBlockIndex)")
	// constraint referenced by the referenced path
	easyfl.Extend("selfReferencedConstraint", "@Path(selfReferencedPath)")

	//--------------------------- constraints ------------------------------------

	easyfl.MustExtendMany(AmountConstraintSource)
	easyfl.MustExtendMany(TimeStampConstraintSource)
	easyfl.MustExtendMany(AddressED25519ConstraintSource)
	easyfl.MustExtendMany(SenderConstraintSource)
	easyfl.MustExtendMany(TimeLockConstraintSource)

	easyfl.PrintLibraryStats()

	initAmountConstraint()
	initTimestampConstraint()
	initAddressED25519Constraint()
	initSenderConstraint()
}

var (
	constraintNameByPrefix = make(map[string]string)
	constraintPrefixByName = make(map[string][]byte)
)

func registerConstraint(name string, prefix []byte) {
	if _, already := constraintPrefixByName[name]; already {
		common.Assert(!already, "repeating constraint name '%s'", name)
	}
	if _, already := constraintNameByPrefix[string(prefix)]; already {
		common.Assert(!already, "repeating constraint prefix %s with name '%s'", easyfl.Fmt(prefix), name)
	}
	common.Assert(0 < len(prefix) && len(prefix) <= 2, "wrong constraint prefix %s, name: %s", easyfl.Fmt(prefix), name)
	constraintNameByPrefix[string(prefix)] = name
	constraintPrefixByName[name] = prefix
}

func ConstraintNameByPrefix(prefix []byte) (string, bool) {
	ret, found := constraintNameByPrefix[string(prefix)]
	return ret, found
}

func ConstraintPrefixByName(name string) ([]byte, bool) {
	ret, found := constraintPrefixByName[name]
	return ret, found
}

func evalPath(ctx *easyfl.CallParams) []byte {
	return ctx.DataContext().(*DataContext).invocationPath
}

func evalAtPath(ctx *easyfl.CallParams) []byte {
	return ctx.DataContext().(*DataContext).dataTree.BytesAtPath(ctx.Arg(0))
}

func evalValidAmount(ctx *easyfl.CallParams) []byte {
	a0 := ctx.Arg(0)
	if len(a0) != 8 {
		return nil
	}
	for _, b := range a0 {
		if b != 0 {
			return []byte{0xff}
		}
	}
	return nil
}

func evalAtArray8(ctx *easyfl.CallParams) []byte {
	arr := lazyslice.ArrayFromBytes(ctx.Arg(0))
	idx := ctx.Arg(1)
	if len(idx) != 1 {
		panic("evalAtArray8: 1-byte value expected")
	}
	return arr.At(int(idx[0]))
}
