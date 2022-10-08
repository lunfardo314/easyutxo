package ledger

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

const OutputIDLength = TransactionIDLength + 1

type OutputID [OutputIDLength]byte

type OutputData []byte

const (
	OutputBlockMasterConstraint = byte(iota)
	OutputBlockTokens
	OutputBlockAddress
	OutputNumRequiredBlocks
)

// Output wraps output lazy tree
type Output struct {
	tree *lazyslice.Tree
}

func NewOutputID(id TransactionID, idx byte) (ret OutputID) {
	copy(ret[:TransactionIDLength], id[:])
	ret[TransactionIDLength] = idx
	return
}

func DummyOutputID() OutputID {
	return NewOutputID(TransactionID{}, 0)
}

func OutputIDFromBytes(data []byte) (ret OutputID, err error) {
	if len(data) != OutputIDLength {
		err = errors.New("OutputIDFromBytes: wrong data length")
		return
	}
	copy(ret[:], data)
	return
}

func (oid *OutputID) TransactionID() TransactionID {
	return oid[:TransactionIDLength]
}

func (oid *OutputID) Index() byte {
	return oid[TransactionIDLength]
}

func (oid *OutputID) Bytes() []byte {
	return oid[:]
}

func (oid *OutputID) String() string {
	return fmt.Sprintf("[%d]%s", oid.Index(), oid.TransactionID())
}

func NewOutput() *Output {
	tr := lazyslice.TreeEmpty()
	for i := 0; i < int(OutputNumRequiredBlocks); i++ {
		tr.PushData(nil, nil)
	}
	// put minimum master constraint
	ret := &Output{tree: tr}
	ret.putMinimumMasterConstraint()
	return ret
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
	o.tree.PutDataAtIdx(OutputBlockAddress, easyutxo.Concat([]byte{constraint}, []byte(addr)), nil)
	o.appendConstraintIndexToTheMasterList(OutputBlockAddress)
}

func (o *Output) AddressConstraint() (Address, byte) {
	ret := o.tree.GetDataAtIdx(OutputBlockAddress, nil)
	return ret[1:], ret[0]
}

func (o *Output) PutTokensConstraint(amount uint64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], amount)
	o.tree.PutDataAtIdx(OutputBlockTokens, easyutxo.Concat([]byte{ConstraintTokens}, b[:]), nil)
	o.appendConstraintIndexToTheMasterList(OutputBlockTokens)
}

func (o *Output) Tokens() uint64 {
	ret := o.tree.GetDataAtIdx(OutputBlockTokens, nil)
	return binary.BigEndian.Uint64(ret[1:])
}

// into the position OutputBlockMasterConstraint puts inline invocation to the `requireAll` with empty list
// later we can append the list with indices of the constraints to be invoked
func (o *Output) putMinimumMasterConstraint() {
	o.tree.PutDataAtIdx(OutputBlockMasterConstraint, []byte{InvocationTypeInline, requireAllCode}, nil)
}

func (o *Output) appendConstraintIndexToTheMasterList(constrIdx byte) {
	constrBin := o.tree.GetDataAtIdx(OutputBlockMasterConstraint, nil)
	constrBin = append(constrBin, constrIdx)
	o.tree.PutDataAtIdx(OutputBlockMasterConstraint, constrBin, nil)
}

func (o *Output) MasterConstraintList() []byte {
	constrBin := o.tree.GetDataAtIdx(OutputBlockMasterConstraint, nil)
	return constrBin[2:]
}
