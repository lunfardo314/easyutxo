package library

import (
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

/*

All integers are treated big-endian. This way lexicographical order coincides with the arithmetic order

The validation context is a tree-like data structure which is validated by evaluating all constraints in it
consumed and produced outputs. The rest of the validation should be done by the logic outside the data itself.
The tree-like data structure is a lazyslice.Array, treated as a tree.

Constants which define validation context data tree branches. Structure of the data tree:

(root)
  -- TransactionBranch = 0x00
       -- TxUnlockParams = 0x00 (path 0x0000)  -- contains unlock params for each input
       -- TxInputIDs = 0x01     (path 0x0001)  -- contains up to 256 inputs, the IDs of consumed outputs
       -- TxOutputBranch = 0x02       (path 0x0002)  -- contains up to 256 produced outputs
       -- TxSignature = 0x03          (path 0x0003)  -- contains the only signature of the essence. It is mandatory
       -- TxTimestamp = 0x04          (path 0x0004)  -- mandatory timestamp of the transaction, Unix seconds
       -- TxInputCommitment = 0x05    (path 0x0005)  -- blake2b hash of the all consumed outputs (which are under path 0x1000)
  -- ConsumedBranch = 0x01
       -- ConsumedOutputsBranch = 0x00 (path 0x0100) -- all consumed outputs, up to 256

All consumed outputs ar contained in the tree element under path 0x0100
A input ID is at path 0x0001ii, where (ii) is 1-byte index of the consumed input in the transaction
This way:
	- the corresponding consumed output is located at path 0x0100ii (replacing 2 byte path prefix with 0x0100)
	- the corresponding unlock-parameters is located at path 0x0000ii (replacing 2 byte path prefix with 0x0000)
*/

// Top level branches
const (
	TransactionBranch = byte(iota)
	ConsumedBranch
)

// Transaction tree
const (
	TxUnlockParams = byte(iota)
	TxInputIDs
	TxOutputs
	TxSignature
	TxTimestamp
	TxInputCommitment
	TxTreeIndexMax
)

const (
	ConsumedOutputsBranch = byte(iota)
)

var (
	PathToConsumedOutputs = lazyslice.Path(ConsumedBranch, ConsumedOutputsBranch)
	PathToProducedOutputs = lazyslice.Path(TransactionBranch, TxOutputs)
	PathToUnlockParams    = lazyslice.Path(TransactionBranch, TxUnlockParams)
	PathToInputIDs        = lazyslice.Path(TransactionBranch, TxInputIDs)
	PathToInputCommitment = lazyslice.Path(TransactionBranch, TxInputCommitment)
	PathToTimestamp       = lazyslice.Path(TransactionBranch, TxTimestamp)
)

// Mandatory output block indices
const (
	OutputBlockAmount = byte(iota)
	OutputBlockTimestamp
	OutputBlockLock
	OutputNumMandatoryBlocks
)

func init() {
	//-------------------------------- standard EasyFL library extensions ------------------------------

	// data context access
	// data context is a lazyslice.Tree
	easyfl.EmbedShort("@", 0, evalPath, true)
	// returns data bytes at the given path of the data context (lazy tree)
	easyfl.EmbedShort("@Path", 1, evalAtPath)

	// gives a vByte cost as big-endian uint16
	easyfl.Extend("#vbCost16", "u16/1")

	// @Array8 interprets $0 as serialized LazyArray with max 256 elements. Takes the $1 element of it. $1 is expected 1-byte long
	easyfl.EmbedLong("@Array8", 2, evalAtArray8)

	// path constants
	easyfl.Extend("pathToTransaction", fmt.Sprintf("%d", TransactionBranch))
	easyfl.Extend("pathToConsumedOutputs", fmt.Sprintf("0x%s", PathToConsumedOutputs.Hex()))
	easyfl.Extend("pathToProducedOutputs", fmt.Sprintf("0x%s", PathToProducedOutputs.Hex()))
	easyfl.Extend("pathToUnlockParams", fmt.Sprintf("0x%s", PathToUnlockParams.Hex()))
	easyfl.Extend("pathToInputIDs", fmt.Sprintf("0x%s", PathToInputIDs.Hex()))
	easyfl.Extend("pathToInputCommitment", fmt.Sprintf("0x%s", PathToInputCommitment.Hex()))
	easyfl.Extend("pathToTimestamp", fmt.Sprintf("0x%s", PathToTimestamp.Hex()))

	// mandatory block indices
	easyfl.Extend("amountBlockIndex", fmt.Sprintf("%d", OutputBlockAmount))
	easyfl.Extend("timestampBlockIndex", fmt.Sprintf("%d", OutputBlockTimestamp))
	easyfl.Extend("lockBlockIndex", fmt.Sprintf("%d", OutputBlockLock))

	// recognize what kind of path
	easyfl.Extend("isPathToConsumedOutput", "hasPrefix($0, pathToConsumedOutputs)")
	easyfl.Extend("isPathToProducedOutput", "hasPrefix($0, pathToProducedOutputs)")

	easyfl.Extend("consumedOutputPathByIndex", "concat(pathToConsumedOutputs,$0)")
	easyfl.Extend("producedOutputPathByIndex", "concat(pathToProducedOutputs,$0)")
	easyfl.Extend("consumedOutputByIndex", "@Path(consumedOutputPathByIndex($0))")
	easyfl.Extend("producedOutputByIndex", "@Path(producedOutputPathByIndex($0))")
	easyfl.Extend("consumedLockByOutputIndex", "@Array8(consumedOutputByIndex($0),lockBlockIndex)")

	// special transaction related

	easyfl.Extend("txBytes", "@Path(pathToTransaction)")
	easyfl.Extend("txID", "blake2b(txBytes)")
	easyfl.Extend("txTimestampBytes", "@Path(pathToTimestamp)")
	easyfl.Extend("txEssenceBytes", "concat(@Path(pathToInputIDs), @Path(pathToProducedOutputs), @Path(pathToInputCommitment))") // timestamp is not a part of the essence

	easyfl.Extend("selfOutputPath", "slice(@,0,2)")
	easyfl.Extend("selfSiblingBlock", "@Array8(@Path(selfOutputPath), $0)")
	easyfl.Extend("selfOutputBytes", "@Path(selfOutputPath)")

	// unlock param branch (0 - transaction, 0 unlock params)
	// invoked output block
	easyfl.Extend("self", "@Path(@)")
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
	easyfl.Extend("selfConstraintData", "constraintData(self)")
	// unlock parameters of the invoked consumed constraint
	easyfl.Extend("selfUnlockParameters", "@Path(concat(pathToUnlockParams, selfConstraintIndex))")
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

// DataContext is the data structure passed to the eval call. It contains:
// - tree: all validation context of the transaction, all data which is to be validated
// - path: a path in the validation context of the constraint being validated in the eval call
type DataContext struct {
	tree *lazyslice.Tree
	path lazyslice.TreePath
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
