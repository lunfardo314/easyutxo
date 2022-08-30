package transaction

import (
	"errors"
	"fmt"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

/*
	Outputs is a lazyslice.Tree
	1st level is pairs: (U, P) where
	- U is script invocation
	- P parameters of the script

	The (U0 || P0) is interpreted as unlock script: the target address. It is used for indexing in the ledger state
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

func NewOutput() *Output {
	return &Output{lazyslice.TreeEmpty()}
}

func OutputFromBytes(data []byte) *Output {
	return &Output{tree: lazyslice.TreeFromBytes(data)}
}

func (o *Output) Bytes() []byte {
	return o.tree.Bytes()
}

func (o *Output) Address() []byte {
	addrType := o.tree.GetDataAtPathAtIdx(0)
	addrData := o.tree.GetDataAtPathAtIdx(1)
	ret := make([]byte, 0, len(addrType)+len(addrData))
	return append(append(ret, addrData...), addrType...)
}
