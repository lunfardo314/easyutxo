package ledger

import (
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

type DataContext struct {
	dataTree       *lazyslice.Tree
	invocationPath lazyslice.TreePath
}

func NewDataContext(tree *lazyslice.Tree, path lazyslice.TreePath, trace ...bool) easyfl.GlobalData {
	var ret easyfl.GlobalData
	if len(trace) > 0 && trace[0] {
		ret = easyfl.NewGlobalDataTracePrint(&DataContext{
			dataTree:       tree,
			invocationPath: path,
		})
	} else {
		ret = easyfl.NewGlobalDataNoTrace(&DataContext{
			dataTree:       tree,
			invocationPath: path,
		})
	}
	return ret
}

var requireAllCode byte

func extendLibrary() {
	// context access
	easyfl.EmbedShort("@", 0, evalPath, true)
	easyfl.EmbedShort("@Path", 1, evalAtPath, true)
	easyfl.Extend("isConsumedBranch", "equal(slice($0,0,1), 0x0100)")
	easyfl.Extend("isProducedBranch", "equal(slice($0,0,1), 0x0002)")

	// LazyArray
	// @Array8 interprets $0 as serialized LazyArray. Takes the $1 element of it. $1 is expected 1-byte
	easyfl.EmbedLong("@Array8", 2, evalAtArray8)

	// special transaction related

	easyfl.Extend("txBytes", "@Path(0x00)")
	easyfl.Extend("txID", "blake2b(txBytes)")
	easyfl.Extend("txInputIDsBytes", "@Path(0x0001)")
	easyfl.Extend("txOutputsBytes", "@Path(0x0002)")
	easyfl.Extend("txTimestampBytes", "@Path(0x0003)")
	easyfl.Extend("txInputCommitmentBytes", "@Path(0x0004)")
	easyfl.Extend("txEssenceBytes", "concat(txInputIDsBytes, txOutputsBytes, txTimestampBytes, txInputCommitmentBytes)")
	easyfl.Extend("addrDataED25519FromPubKey", "blake2b($0)")

	easyfl.Extend("selfOutputBytes", "@Path(slice(@,0,2))")
	easyfl.Extend("selfBlockBytes", "@Array8(selfOutputBytes, $0)")

	easyfl.Extend("selfConstraint", "@Path(@)")
	easyfl.Extend("selfConstraintData", "tail(selfConstraint,1)")
	easyfl.Extend("selfOutputIndex", "tail(@,2)")
	easyfl.Extend("selfUnlockBlock", "@Path(concat(0, 0, slice(@, 2, 3)))")
	easyfl.Extend("selfReferencedConstraint", "@Path(concat(slice(@,0,1), selfUnlockBlock))")

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
