package ledger

import (
	"bytes"
	ed255192 "crypto/ed25519"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"golang.org/x/crypto/blake2b"
)

// ValidationContext is a data structure, which contains transaction, consumed outputs and constraint library
type ValidationContext struct {
	tree  *lazyslice.Tree
	trace bool
}

var Path = lazyslice.Path

// Top level branches
const (
	TransactionBranch = byte(iota)
	ConsumedContextBranch
)

const (
	ConsumedContextOutputsBranch = byte(iota)
)

// Invocation types are indices of constraints in the global library
const (
	InvocationTypeInline = byte(iota)
	InvocationTypeUnlockScript
)

// ValidationContextFromTransaction constructs lazytree from transaction bytes and consumed outputs
func ValidationContextFromTransaction(txBytes []byte, ledgerState StateAccess, trace ...bool) (*ValidationContext, error) {
	txBranch := lazyslice.ArrayFromBytes(txBytes, int(TxTreeIndexMax))
	inputIDs := lazyslice.ArrayFromBytes(txBranch.At(int(TxInputIDsBranch)), 256)

	var err error
	var oid OutputID

	consumedOutputsArray := lazyslice.EmptyArray(256)
	inputIDs.ForEach(func(i int, data []byte) bool {
		if oid, err = OutputIDFromBytes(data); err != nil {
			return false
		}
		od, ok := ledgerState.GetUTXO(&oid)
		if !ok {
			err = fmt.Errorf("input not found: %s", oid.String())
			return false
		}
		consumedOutputsArray.Push(od)
		return true
	})
	if err != nil {
		return nil, err
	}
	ctx := lazyslice.MakeArray(
		txBytes, // TransactionBranch = 0
		lazyslice.MakeArray(consumedOutputsArray), // ConsumedContextBranch = 1
	)
	ret := &ValidationContext{
		tree:  ctx.AsTree(),
		trace: len(trace) > 0 && trace[0],
	}
	return ret, nil
}

func (v *ValidationContext) parseInvocationCode(invocationFullPath lazyslice.TreePath) ([]byte, string) {
	invocation := v.tree.BytesAtPath(invocationFullPath)
	common.Assert(len(invocation) >= 1, "constraint can't be empty")
	switch invocation[0] {
	case InvocationTypeUnlockScript:
		// unlock block must provide script which is pre-image of the hash
		scriptBinary := v.unlockScriptBinary(invocationFullPath)
		h := blake2b.Sum256(scriptBinary)
		common.Assert(bytes.Equal(h[:], invocation[1:]), "wrong script")
		return invocation[1:], "(unlock code by hash)"
	case InvocationTypeInline:
		return invocation[1:], "(in-line constraint)"
	}
	return mustGetConstraintBinary(invocation[0])
}

func (v *ValidationContext) unlockScriptBinary(invocationFullPath lazyslice.TreePath) []byte {
	unlockBlockPath := easyutxo.Concat(invocationFullPath)
	unlockBlockPath[1] = TxUnlockParamsBranch
	return v.tree.BytesAtPath(unlockBlockPath)
}

func (v *ValidationContext) rootContext() easyfl.GlobalData {
	return NewDataContext(v.tree, nil)
}

func (v *ValidationContext) TransactionBytes() []byte {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txBytes")
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *ValidationContext) TransactionEssenceBytes() []byte {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txEssenceBytes")
	if err != nil {
		panic(err)
	}
	return ret
}

func (v *ValidationContext) TransactionID() TransactionID {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txID")
	if err != nil {
		panic(err)
	}
	var txid TransactionID
	common.Assert(len(txid[:]) == len(ret), "wrong data length")
	copy(txid[:], ret)
	return txid
}

func (v *ValidationContext) InputCommitment() []byte {
	return v.tree.BytesAtPath(Path(TransactionBranch, TxInputCommitment))
}

func (v *ValidationContext) UnlockED25519Inputs(pairs []*keyPair) {
	_ = prepareKeyPairs(pairs)
	// TODO
}

type keyPair struct {
	pubKey  ed255192.PublicKey
	privKey ed255192.PrivateKey
}

func prepareKeyPairs(keyPairs []*keyPair) map[string]*keyPair {
	ret := make(map[string]*keyPair)
	for _, kp := range keyPairs {
		addr := AddressFromED25519PubKey(kp.pubKey)
		ret[string(addr.Bytes())] = kp
	}
	return ret
}

func (v *ValidationContext) ConsumedOutput(idx byte) *Output {
	ret, err := OutputFromBytes(v.tree.BytesAtPath(Path(ConsumedContextBranch, ConsumedContextOutputsBranch, idx)))
	common.AssertNoError(err)
	return ret
}

func (v *ValidationContext) ForEachOutput(branch lazyslice.TreePath, fun func(out *Output, path lazyslice.TreePath) bool) {
	outputPath := Path(branch, byte(0))
	v.tree.ForEach(func(idx byte, outputData []byte) bool {
		outputPath[2] = idx
		out, err := OutputFromBytes(outputData)
		common.AssertNoError(err)
		return fun(out, outputPath)
	}, branch)
}

func (v *ValidationContext) ForEachInputID(fun func(idx byte, oid *OutputID) bool) {
	v.tree.ForEach(func(i byte, data []byte) bool {
		oid, err := OutputIDFromBytes(data)
		common.AssertNoError(err)
		if !fun(i, &oid) {
			return false
		}
		return true
	}, Path(TransactionBranch, TxInputIDsBranch))
}
