package ledger

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/lunfardo314/easyutxo/lazyslice"
)

const OutputIDLength = IDLength + 2

type OutputID [OutputIDLength]byte

type OutputData []byte

const (
	OutputBlockValidationScripts = byte(iota)
	OutputBlockAddress
	OutputBlockTokens
	OutputNumRequiredBlocks
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

func NewOutput() *Output {
	tr := lazyslice.TreeEmpty()
	for i := 0; i < int(OutputNumRequiredBlocks); i++ {
		tr.PushData(nil, nil)
	}
	return &Output{tree: tr}
}

func OutputFromBytes(data []byte) (*Output, error) {
	ret := &Output{tree: lazyslice.TreeFromBytes(data)}
	if ret.tree.NumElements(nil) < int(OutputNumRequiredBlocks) {
		return nil, fmt.Errorf("output is expected to have at least %d blocks", OutputNumRequiredBlocks)
	}
	return ret, nil
}

func OutputFromTree(tree *lazyslice.Tree) *Output {
	return &Output{tree}
}

func (o *Output) Bytes() []byte {
	return o.tree.Bytes()
}

func (o *Output) PutAddressConstraint(addr Address, constraint byte) {
	var buf bytes.Buffer
	buf.WriteByte(constraint)
	buf.Write(addr)
	o.tree.PutDataAtIdx(OutputBlockAddress, buf.Bytes(), nil)
}

func (o *Output) AddressConstraint() (Address, byte) {
	ret := o.tree.GetDataAtIdx(OutputBlockAddress, nil)
	return ret[1:], ret[0]
}

func (o *Output) PutTokens(amount uint64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], amount)
	o.tree.PutDataAtIdx(OutputBlockTokens, b[:], nil)
}

func (o *Output) Tokens() uint64 {
	ret := o.tree.GetDataAtIdx(OutputBlockTokens, nil)
	if len(ret) != 8 {
		panic("tokens must be 8 bytes")
	}
	return binary.BigEndian.Uint64(ret)
}
