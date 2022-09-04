package ledger

import (
	"errors"
	"fmt"

	"github.com/lunfardo314/easyutxo/lazyslice"
)

const OutputIDLength = IDLength + 2

type OutputID [OutputIDLength]byte

type OutputData []byte

const (
	OutputIndexValidationScripts = byte(iota)
	OutputIndexAddress
)

// Output represents output (UTXO) in the ledger
type Output struct {
	tree *lazyslice.Tree
}

func NewOutputID(id ID, outputGroup byte, indexInGroup byte) (ret OutputID) {
	copy(ret[:IDLength], id[:])
	ret[IDLength] = outputGroup
	ret[IDLength+1] = indexInGroup
	return
}

func OutputIDFromBytes(data []byte) (ret OutputID, err error) {
	if len(data) != OutputIDLength {
		err = errors.New("OutputIDFromBytes: wrong data length")
		return
	}
	copy(ret[:], data)
	return
}

func (oid *OutputID) Parts() (txid ID, group, index byte) {
	copy(txid[:], oid[:IDLength])
	group = oid[IDLength]
	index = oid[IDLength+1]
	return
}

func (oid *OutputID) String() string {
	txid, group, idx := oid.Parts()
	return fmt.Sprintf("[%d|%d]%s", group, idx, txid.String())
}

func OutputFromBytes(data []byte) *Output {
	return &Output{tree: lazyslice.TreeFromBytes(data)}
}

func OutputFromTree(tree *lazyslice.Tree) *Output {
	return &Output{tree}
}

func (o *Output) Bytes() []byte {
	return o.tree.Bytes()
}

func (o *Output) Address() []byte {
	return o.tree.GetDataAtIdx(OutputIndexAddress, nil)
}

// ValidateOutput invokes all invokable scripts in the output in the context of ledger (not input)
func (v *ValidationContext) ValidateOutput(outputContext, idx byte) {
	o := v.Output(outputContext, idx)
	invocationList := o.tree.GetDataAtIdx(0, nil)
	for _, invokeAtIdx := range invocationList {
		v.RunScript(Path(ValidationCtxTransactionIndex, TxTreeIndexOutputGroups, outputContext, idx, invokeAtIdx))
	}
}

// ValidateOutputs traverses all outputs in the ledger and validates each
func (v *ValidationContext) ValidateOutputs() {

}
