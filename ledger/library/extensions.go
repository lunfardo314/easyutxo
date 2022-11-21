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
	TxLocalLibraries
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
	PathToSignature       = lazyslice.Path(TransactionBranch, TxSignature)
	PathToInputCommitment = lazyslice.Path(TransactionBranch, TxInputCommitment)
	PathToLocalLibrary    = lazyslice.Path(TransactionBranch, TxLocalLibraries)
	PathToTimestamp       = lazyslice.Path(TransactionBranch, TxTimestamp)
)

// Mandatory output block indices
const (
	ConstraintIndexAmount = byte(iota)
	ConstraintIndexTimestamp
	ConstraintIndexLock
)

func init() {
	//-------------------------------- standard EasyFL library extensions ------------------------------

	// data context access
	// data context is a lazyslice.Tree
	easyfl.EmbedShort("@", 0, evalPath, true)
	// returns data bytes at the given path of the data context (lazy tree)
	easyfl.EmbedShort("@Path", 1, evalAtPath)

	// gives a vByte cost as big-endian uint16
	easyfl.Extend("vbCost16", "u16/1")

	// @Array8 interprets $0 as serialized LazyArray with max 256 elements. Takes the $1 element of it. $1 is expected 1-byte long
	easyfl.EmbedLong("@Array8", 2, evalAtArray8)

	easyfl.EmbedLong("callLocalLibrary", -1, evalCallLocalLibrary)

	// path constants
	easyfl.Extend("pathToTransaction", fmt.Sprintf("%d", TransactionBranch))
	easyfl.Extend("pathToConsumedOutputs", fmt.Sprintf("0x%s", PathToConsumedOutputs.Hex()))
	easyfl.Extend("pathToProducedOutputs", fmt.Sprintf("0x%s", PathToProducedOutputs.Hex()))
	easyfl.Extend("pathToUnlockParams", fmt.Sprintf("0x%s", PathToUnlockParams.Hex()))
	easyfl.Extend("pathToInputIDs", fmt.Sprintf("0x%s", PathToInputIDs.Hex()))
	easyfl.Extend("pathToSignature", fmt.Sprintf("0x%s", PathToSignature.Hex()))
	easyfl.Extend("pathToInputCommitment", fmt.Sprintf("0x%s", PathToInputCommitment.Hex()))
	easyfl.Extend("pathToLocalLibrary", fmt.Sprintf("0x%s", PathToLocalLibrary.Hex()))
	easyfl.Extend("pathToTimestamp", fmt.Sprintf("0x%s", PathToTimestamp.Hex()))

	// mandatory block indices in the output
	easyfl.Extend("amountBlockIndex", fmt.Sprintf("%d", ConstraintIndexAmount))
	easyfl.Extend("timestampBlockIndex", fmt.Sprintf("%d", ConstraintIndexTimestamp))
	easyfl.Extend("lockBlockIndex", fmt.Sprintf("%d", ConstraintIndexLock))

	// mandatory constraints and values
	// $0 is output binary as lazy array
	easyfl.Extend("amountConstraint", "@Array8($0, amountBlockIndex)")
	easyfl.Extend("timestampConstraint", "@Array8($0, timestampBlockIndex)")
	easyfl.Extend("lockConstraint", "@Array8($0, lockBlockIndex)")

	// recognize what kind of path is at $0
	easyfl.Extend("isPathToConsumedOutput", "hasPrefix($0, pathToConsumedOutputs)")
	easyfl.Extend("isPathToProducedOutput", "hasPrefix($0, pathToProducedOutputs)")

	// make branch path by index $0
	easyfl.Extend("consumedOutputPathByIndex", "concat(pathToConsumedOutputs,$0)")
	easyfl.Extend("unlockParamsPathByIndex", "concat(pathToUnlockParams,$0)")
	easyfl.Extend("producedOutputPathByIndex", "concat(pathToProducedOutputs,$0)")

	// takes 1-byte $0 as output index
	easyfl.Extend("consumedOutputByIndex", "@Path(consumedOutputPathByIndex($0))")
	easyfl.Extend("unlockParamsByIndex", "@Path(unlockParamsPathByIndex($0))")
	easyfl.Extend("producedOutputByIndex", "@Path(producedOutputPathByIndex($0))")

	// takes $0 'constraint index' as 2 bytes: 0 for output index, 1 for block index
	easyfl.Extend("producedConstraintByIndex", "@Array8(producedOutputByIndex(byte($0,0)), byte($0,1))")
	easyfl.Extend("consumedConstraintByIndex", "@Array8(consumedOutputByIndex(byte($0,0)), byte($0,1))")
	easyfl.Extend("unlockParamsByConstraintIndex", "@Array8(unlockParamsByIndex(byte($0,0)), byte($0,1))")

	easyfl.Extend("consumedLockByOutputIndex", "consumedConstraintByIndex(concat($0, lockBlockIndex))")

	easyfl.Extend("inputIDByIndex", "@Path(concat(pathToInputIDs,$0))")

	// special transaction related

	easyfl.Extend("txBytes", "@Path(pathToTransaction)")
	easyfl.Extend("txID", "blake2b(txBytes)")
	easyfl.Extend("txSignature", "@Path(pathToSignature)")
	easyfl.Extend("txTimestampBytes", "@Path(pathToTimestamp)")
	easyfl.Extend("txEssenceBytes", "concat(@Path(pathToInputIDs), @Path(pathToProducedOutputs), @Path(pathToInputCommitment))") // timestamp is not a part of the essence

	// functions with prefix 'self' are invocation context specific, i.e. they use function '@' to calculate
	// local values which depend on the invoked constraint

	easyfl.Extend("selfOutputPath", "slice(@,0,2)")
	easyfl.Extend("selfSiblingConstraint", "@Array8(@Path(selfOutputPath), $0)")
	easyfl.Extend("selfOutputBytes", "@Path(selfOutputPath)")

	// unlock param branch (0 - transaction, 0 unlock params)
	// invoked output block
	easyfl.Extend("self", "@Path(@)")
	// bytecode prefix of the invoked constraint
	easyfl.Extend("selfBytecodePrefix", "parseBytecodePrefix(self)")

	easyfl.Extend("selfIsConsumedOutput", "isPathToConsumedOutput(@)")
	easyfl.Extend("selfIsProducedOutput", "isPathToProducedOutput(@)")

	// output index of the invocation
	easyfl.Extend("selfOutputIndex", "byte(@, 2)")
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
	// returns unlock block of the sibling
	easyfl.Extend("selfSiblingUnlockBlock", "@Array8(@Path(concat(pathToUnlockParams, selfOutputIndex)), $0)")

	// returns selfUnlockParameters if blake2b hash of it is equal to the given hash, otherwise nil
	easyfl.Extend("selfHashUnlock", "if(equal($0, blake2b(selfUnlockParameters)),selfUnlockParameters,nil)")

	// takes ED25519 signature from full signature, first 64 bytes
	easyfl.Extend("signatureED25519", "slice($0, 0, 63)")
	// takes ED25519 public key from full signature
	easyfl.Extend("publicKeyED25519", "slice($0, 64, 95)")

	// init constraints
	initAmountConstraint()
	initTimestampConstraint()
	initAddressED25519Constraint()
	initDeadlineLockConstraint()
	initTimelockConstraint()
	initSenderConstraint()
	initChainConstraint()
	initChainLockConstraint()
	initChainRoyaltiesConstraint()
	initImmutableConstraint()
	initCommitToSiblingConstraint()

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

// CompileLocalLibrary compiles local library and serializes it as lazy array
func CompileLocalLibrary(source string) ([]byte, error) {
	libBin, err := easyfl.CompileLocalLibrary(source)
	if err != nil {
		return nil, err
	}
	ret := lazyslice.MakeArrayFromData(libBin...)
	return ret.Bytes(), nil
}

// arg 0 - local library binary (as lazy array)
// arg 1 - 1-byte index of then function in the library
// arg 2 ... arg 15 optional arguments
func evalCallLocalLibrary(ctx *easyfl.CallParams) []byte {
	arr := lazyslice.ArrayFromBytes(ctx.Arg(0))
	libData := arr.Parsed()
	idx := ctx.Arg(1)
	if len(idx) != 1 || int(idx[0]) >= len(libData) {
		ctx.TracePanic("evalCallLocalLibrary: wrong function index")
	}
	ret := easyfl.CallLocalLibrary(ctx.Slice(2, ctx.Arity()), libData, int(idx[0]))
	ctx.Trace("evalCallLocalLibrary: lib#%d -> %s", idx[0], easyfl.Fmt(ret))
	return ret
}
