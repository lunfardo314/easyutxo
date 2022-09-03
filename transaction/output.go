package transaction

import (
	"errors"
	"fmt"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

/*
	Outputs is a lazyslice.Tree
	1st level:
	- at index 0 is index of validation scripts in the output. The byte value > 0 points to another element which is script
	- all other from index 1 are data. Those which are in the index, should be invocations of scripts. The rest is just data
	- first byte in each script invocation points to the element in the script library.
	- The library element is a script which interprets the rest of the invocation
*/

const OutputIDLength = IDLength + 2

type OutputID [OutputIDLength]byte

type OutputData []byte

func NewOutputID(id ID, outputIndex uint16) (ret OutputID) {
	copy(ret[:IDLength], id[:])
	copy(ret[IDLength:], easyutxo.EncodeInteger(outputIndex))
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

func (oid *OutputID) Parts() (txid ID, index uint16) {
	copy(txid[:], oid[:IDLength])
	index = easyutxo.DecodeInteger[uint16](oid[IDLength:])
	return
}

func (oid *OutputID) String() string {
	txid, idx := oid.Parts()
	return fmt.Sprintf("[%d]%s", idx, txid.String())
}

type Output struct {
	tree *lazyslice.Tree
}

func OutputFromBytes(data []byte) *Output {
	return &Output{tree: lazyslice.TreeFromBytes(data)}
}

func (o *Output) Tree() *lazyslice.Tree {
	return o.tree
}

func (o *Output) Bytes() []byte {
	return o.tree.Bytes()
}

func (o *Output) Address() []byte {
	addrType := o.tree.GetDataAtIdx(0, nil)
	addrData := o.tree.GetDataAtIdx(1, nil)
	ret := make([]byte, 0, len(addrType)+len(addrData))
	return append(append(ret, addrData...), addrType...)
}

func (v *ValidationContext) ValidateOutput(outputContext, idx byte) {
	o := v.Output(outputContext, idx)
	if o.tree.NumElements(nil)%2 != 0 {
		panic("number of elements in the output must be even")
	}
	for i := 0; i < o.tree.NumElements(nil)%2; i++ {
		engine.Run(v.Tree(), Path(ValidationCtxTxIndex, TxTreeIndexOutputsLong, outputContext, idx))
	}
}

func (v *ValidationContext) ValidateOutputs() {
	numOutputs := v.tree.NumElementsLong(Path(ValidationCtxTxIndex, TxTreeIndexOutputsLong))
	for i := 0; i < numOutputs; i++ {
		idxBin := easyutxo.EncodeInteger(uint16(i))
		v.ValidateOutput(idxBin[0], idxBin[1])
	}
}
