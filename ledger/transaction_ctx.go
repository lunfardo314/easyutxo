package ledger

import (
	ed255192 "crypto/ed25519"
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

// ValidationContext is a data structure, which contains transaction, consumed outputs and constraint library
type ValidationContext struct {
	tree *lazyslice.Tree
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

// TransactionBranch. 1st level branches
const (
	TxUnlockParamsBranch = byte(iota)
	TxInputIDsBranch
	TxOutputBranch
	TxTimestamp
	TxInputCommitment
	TxTreeIndexMax
)

// Invocation types are indices of constraints in the global library
const (
	InvocationTypeInline = byte(iota)
	InvocationTypeUnlockScript
)

func NewValidationContext() *ValidationContext {
	tx := NewTransaction()
	ret, err := ValidationContextFromTransaction(tx.Bytes(), nil)
	easyutxo.AssertNoError(err)
	return ret
}

// ValidationContextFromTransaction constructs lazytree from transaction bytes and consumed outputs
func ValidationContextFromTransaction(txBytes []byte, ledgerState StateAccess) (*ValidationContext, error) {
	txBranch := lazyslice.ArrayFromBytes(txBytes, int(TxTreeIndexMax))
	inputIDs := lazyslice.ArrayFromBytes(txBranch.At(int(TxInputIDsBranch)))

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
	consumedBranch := lazyslice.EmptyArray(1)
	consumedBranch.Push(consumedOutputsArray.Bytes())
	ctx := lazyslice.EmptyArray(2)
	ctx.Push(consumedBranch.Bytes())
	ctx.Push(txBytes)
	ret := &ValidationContext{tree: ctx.AsTree()}
	return ret, nil
}

func TransactionContextFromTree(dataTree *lazyslice.Tree) *ValidationContext {
	return &ValidationContext{
		tree: dataTree,
	}
}

func invokeConstraint(tree *lazyslice.Tree, path lazyslice.TreePath) []byte {
	return TransactionContextFromTree(tree).Invoke(path)
}

func (v *ValidationContext) Tree() *lazyslice.Tree {
	return v.tree
}

func (v *ValidationContext) Transaction() *Transaction {
	return TransactionFromBytes(v.tree.BytesAtPath(Path(TransactionBranch)))
}

func (v *ValidationContext) CodeFromGlobalLibrary(idx byte) []byte {
	return v.tree.BytesAtPath(Path(ConsumedContextBranch, ConsumedContextConstraintLibraryBranch, idx))
}

func (v *ValidationContext) CodeFromLocalLibrary(idx byte) []byte {
	return v.tree.BytesAtPath(Path(TransactionBranch, TxLocalLibraryBranch, idx))
}

func (v *ValidationContext) parseInvocationCode(invocationFullPath lazyslice.TreePath) []byte {
	invocation := v.tree.BytesAtPath(invocationFullPath)
	if len(invocation) < 1 {
		panic("empty invocation")
	}
	switch invocation[0] {
	case InvocationTypeUnlockScript:
		if len(invocation) < 2 {
			panic("wrong invocation")
		}
		return v.CodeFromLocalLibrary(invocation[1])
	case InvocationTypeInline:
		return invocation[1:]
	}
	return v.CodeFromGlobalLibrary(invocation[0])
}

func (v *ValidationContext) rootContext() *DataContext {
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

func (v *ValidationContext) TransactionID() []byte {
	ret, err := easyfl.EvalFromSource(v.rootContext(), "txID")
	if err != nil {
		panic(err)
	}
	return ret
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
	return OutputFromBytes(v.tree.BytesAtPath(Path(ConsumedContextBranch, ConsumedContextOutputsBranch, idx)))
}

func (v *ValidationContext) ForEachOutput(branch lazyslice.TreePath, fun func(out *Output, path lazyslice.TreePath) bool) {
	outputPath := Path(branch, byte(0))
	v.tree.ForEach(func(idx byte, outputData []byte) bool {
		outputPath[2] = idx
		return fun(OutputFromBytes(outputData), outputPath)
	}, branch)
}
