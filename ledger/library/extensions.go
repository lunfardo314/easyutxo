package library

import (
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

type DataContext struct {
	tree *lazyslice.Tree
	path lazyslice.TreePath
	// TODO caching
}

func NewDataContext(tree *lazyslice.Tree) *DataContext {
	return &DataContext{tree: tree}
}

func (c *DataContext) DataTree() *lazyslice.Tree {
	return c.tree
}

func (c *DataContext) Path() lazyslice.TreePath {
	return c.path
}

func (c *DataContext) SetPath(path lazyslice.TreePath) {
	c.path = easyfl.Concat(path.Bytes())
}

func init() {
	//-------------------------------- standard EasyFL library extensions ------------------------------

	// context access
	easyfl.EmbedShort("@", 0, evalPath, true)
	easyfl.EmbedShort("@Path", 1, evalAtPath, true)
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
	easyfl.Extend("txEssenceBytes", "concat(txInputIDsBytes, txOutputsBytes, txInputCommitmentBytes)") // timestamp is not a part of the essence

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

	// init constraints
	initAmountConstraint()
	initTimestampConstraint()
	initAddressED25519Constraint()
	initTimelockConstraint()
	//initSenderConstraint()
	initDeadlineLockConstraint()

	easyfl.PrintLibraryStats()
}

func evalPath(ctx *easyfl.CallParams) []byte {
	return ctx.DataContext().(*DataContext).Path()
}

func evalAtPath(ctx *easyfl.CallParams) []byte {
	return ctx.DataContext().(*DataContext).DataTree().BytesAtPath(ctx.Arg(0))
}

func evalAtArray8(ctx *easyfl.CallParams) []byte {
	arr := lazyslice.ArrayFromBytes(ctx.Arg(0))
	idx := ctx.Arg(1)
	if len(idx) != 1 {
		panic("evalAtArray8: 1-byte value expected")
	}
	return arr.At(int(idx[0]))
}
