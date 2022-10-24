package ledger

import (
	"bytes"
	ed255192 "crypto/ed25519"
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"golang.org/x/crypto/blake2b"
)

// ValidationContext is a data structure, which contains transaction, consumed outputs and constraint library
type ValidationContext struct {
	tree        *lazyslice.Tree
	traceOption int
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

const (
	TraceOptionNone = iota
	TraceOptionAll
	TraceOptionFailedConstraints
)

// ValidationContextFromTransaction constructs lazytree from transaction bytes and consumed outputs
func ValidationContextFromTransaction(txBytes []byte, ledgerState StateAccess, traceOption ...int) (*ValidationContext, error) {
	txBranch := lazyslice.ArrayFromBytes(txBytes, int(TxTreeIndexMax))
	inputIDs := lazyslice.ArrayFromBytes(txBranch.At(int(TxInputIDsBranch)), 256)

	var err error
	var oid OutputID

	consumedOutputsArray := lazyslice.EmptyArray(256)
	ids := make(map[string]struct{})
	inputIDs.ForEach(func(i int, data []byte) bool {
		if oid, err = OutputIDFromBytes(data); err != nil {
			return false
		}
		// check repeating inputIDs
		if _, repeating := ids[string(data)]; repeating {
			err = fmt.Errorf("repeating input ID: %s", oid.String())
			return false
		}
		ids[string(data)] = struct{}{}

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
		tree:        ctx.AsTree(),
		traceOption: TraceOptionNone,
	}
	if len(traceOption) > 0 {
		ret.traceOption = traceOption[0]
	}
	return ret, nil
}

const (
	unlockScriptName = "(unlock script constraint)"
	inlineScriptName = "(in-line script constraint)"
)

// parseInvocationScript return binary script, name for tracing and flag if it is tu be run (true) or ignored (false)
func (v *ValidationContext) parseInvocationScript(invocationFullPath lazyslice.TreePath) ([]byte, string, bool) {
	invocation := v.tree.BytesAtPath(invocationFullPath)
	easyfl.Assert(len(invocation) >= 1, "constraint can't be empty")
	producedOutputContext := bytes.HasPrefix(invocationFullPath, []byte{TransactionBranch, TxOutputBranch})
	switch ConstraintType(invocation[0]) {
	case ConstraintTypeUnlockScript:
		if producedOutputContext {
			return nil, unlockScriptName, false
		}
		// unlock block must provide script which is pre-image of the hash
		scriptBinary := v.unlockScriptBinary(invocationFullPath)
		h := blake2b.Sum256(scriptBinary)
		easyfl.Assert(bytes.Equal(h[:], invocation[1:]), "script does not match provided hash")
		return invocation[1:], unlockScriptName, true

	case ConstraintTypeInlineScript:
		if producedOutputContext {
			return nil, inlineScriptName, false
		}
		return invocation[1:], inlineScriptName, true
	}
	script, name := mustGetConstraintBinary(ConstraintType(invocation[0]))
	return script, name, true
}

// unlockScriptBinary finds script from unlock block
func (v *ValidationContext) unlockScriptBinary(invocationFullPath lazyslice.TreePath) []byte {
	unlockBlockPath := easyfl.Concat(invocationFullPath)
	unlockBlockPath[1] = TxUnlockParamsBranch
	return v.tree.BytesAtPath(unlockBlockPath)
}

func (v *ValidationContext) rootContext() easyfl.GlobalData {
	return v.dataContext(nil)
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
	easyfl.Assert(len(txid[:]) == len(ret), "wrong data length")
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
		addr := LockFromED25519PubKey(kp.pubKey)
		ret[string(addr.Bytes())] = kp
	}
	return ret
}

func (v *ValidationContext) ConsumedOutput(idx byte) *Output {
	ret, err := OutputFromBytes(v.tree.BytesAtPath(Path(ConsumedContextBranch, ConsumedContextOutputsBranch, idx)))
	easyfl.AssertNoError(err)
	return ret
}

func (v *ValidationContext) ForEachOutput(branch lazyslice.TreePath, fun func(out *Output, path lazyslice.TreePath) bool) {
	outputPath := easyfl.Concat(branch, byte(0))
	v.tree.ForEach(func(idx byte, outputData []byte) bool {
		outputPath[2] = idx
		out, err := OutputFromBytes(outputData)
		easyfl.AssertNoError(err)
		return fun(out, outputPath)
	}, branch)
}

func (v *ValidationContext) ForEachInputID(fun func(idx byte, oid *OutputID) bool) {
	v.tree.ForEach(func(i byte, data []byte) bool {
		oid, err := OutputIDFromBytes(data)
		easyfl.AssertNoError(err)
		if !fun(i, &oid) {
			return false
		}
		return true
	}, Path(TransactionBranch, TxInputIDsBranch))
}
