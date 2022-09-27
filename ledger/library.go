package ledger

import (
	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

type DataContext struct {
	dataTree       *lazyslice.Tree
	invocationPath lazyslice.TreePath
}

func NewDataContext(tree *lazyslice.Tree, path lazyslice.TreePath) *DataContext {
	return &DataContext{
		dataTree:       tree,
		invocationPath: path,
	}
}

func extendLibrary() {
	// context access
	easyfl.EmbedShort("@", 0, evalPath)
	easyfl.EmbedShort("atPath", 1, evalAtPath)
	// special transaction related

	easyfl.Extend("txBytes", "atPath(0x00)")
	easyfl.Extend("txInputIDsBytes", "atPath(0x0001)")
	easyfl.Extend("txOutputBytes", "atPath(0x0002)")
	easyfl.Extend("txTimestampBytes", "atPath(0x0003)")
	easyfl.Extend("txInputCommitmentBytes", "atPath(0x0004)")
	easyfl.Extend("txLocalLibBytes", "atPath(0x0005)")
	easyfl.Extend("txid", "blake2b(txBytes)")
	easyfl.Extend("txEssenceBytes", "concat(txInputIDsBytes, txOutputBytes, txTimestampBytes, txInputCommitmentBytes, txLocalLibBytes)")
	easyfl.Extend("addrED25519FromPubKey", "blake2b($0)")

	easyfl.Extend("selfConstraint", "atPath(@)")
	easyfl.Extend("selfConstraintData", "if(equal(slice(selfConstraint,0,1), 0),nil,tail(selfConstraint,1))")
	easyfl.Extend("selfOutputIndex", "slice(@,2,4)")
	easyfl.Extend("selfUnlockBlock", "atPath(concat(0, 0, slice(@, 2, 5)))")
	easyfl.Extend("selfReferencedConstraint", "atPath(concat(slice(@,0,2), selfUnlockBlock))")
	easyfl.Extend("selfConsumedContext", "equal(slice(@,0,2), 0x0100)")
	easyfl.Extend("selfOutputContext", "not(selfConsumedContext)")
}

func evalPath(ctx *easyfl.CallParams) []byte {
	return ctx.DataContext().(*DataContext).invocationPath
}

func evalAtPath(ctx *easyfl.CallParams) []byte {
	return ctx.DataContext().(*DataContext).dataTree.BytesAtPath(ctx.Arg(0))
}

func evalRequireAll(ctx *easyfl.CallParams) []byte {
	blockIndices := ctx.Arg(0)
	path := ctx.DataContext().(*DataContext).invocationPath
	myIdx := path[len(path)-1]
	pathCopy := make([]byte, len(path))
	copy(pathCopy, path)

	for _, idx := range blockIndices {
		if idx <= myIdx {
			// only forward
			panic("evalRequireAll: can only invoke constraints forward")
		}
		pathCopy[len(path)-1] = idx
		if len(invokeConstraint(ctx.DataContext().(*DataContext).dataTree, path)) == 0 {
			return nil
		}
	}
	return []byte{0xff}
}

func evalRequireAny(ctx *easyfl.CallParams) []byte {
	blockIndices := ctx.Arg(0)
	gc := ctx.DataContext().(*DataContext)
	path := gc.invocationPath
	myIdx := path[len(path)-1]
	pathCopy := make([]byte, len(path))
	copy(pathCopy, path)

	for _, idx := range blockIndices {
		if idx <= myIdx {
			// only forward
			panic("evalRequireAll: can only invoke constraints forward")
		}
		path[len(path)-1] = idx
		if len(invokeConstraint(gc.dataTree, path)) != 0 {
			return []byte{0xff}
		}
	}
	return nil
}
