package ledger

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

// Output is a LazyArray with constraint invocations
// There must be at least 2 constraints:
// - at index 0 it is 'main constraint'
// - at index 1 it is 'address constraint'
// The main constraint has:
// ---  constraint code 0 (byte 0). It checks basic constraints for amount and timestamp
// ---  data is: 4 bytes of timestamp and 8 bytes of amount.
// The address constraint is any constraint invocation. The block is treated as 'address'
// and it is used as index value for the search of 'all UTXOs belonging to the address'

const OutputIDLength = TransactionIDLength + 1

type OutputID [OutputIDLength]byte

type OutputData []byte

const (
	OutputBlockMain = byte(iota)
	OutputBlockAddress
	OutputNumRequiredBlocks
)

// Output wraps output lazy blocks
type Output struct {
	blocks *lazyslice.Array
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
	ret := &Output{
		blocks: lazyslice.EmptyArray(256),
	}
	ret.blocks.PushEmptyElements(2)
	return ret
}

func OutputFromBytes(data []byte) *Output {
	return &Output{
		blocks: lazyslice.ArrayFromBytes(data, 256),
	}
}

func (o *Output) Bytes() []byte {
	return o.blocks.Bytes()
}

func (o *Output) BlockBytes(idx byte) []byte {
	return o.blocks.At(int(idx))
}

func (o *Output) PutAddress(addr AddressData, constraint byte) {
	o.blocks.PutAtIdx(OutputBlockAddress, easyutxo.Concat(constraint, []byte(addr)))
}

func (o *Output) Address() []byte {
	return o.blocks.At(int(OutputBlockAddress))
}

func (o *Output) PutMainConstraint(timestamp uint32, amount uint64) {
	var a [8]byte
	var ts [4]byte
	binary.BigEndian.PutUint64(a[:], amount)
	binary.BigEndian.PutUint32(ts[:], timestamp)

	o.blocks.PutAtIdx(OutputBlockMain, easyutxo.Concat(ConstraintMain, ts[:], a[:]))
}

func (o *Output) Timestamp() uint32 {
	mainBlock := o.blocks.At(int(OutputBlockMain))
	return binary.BigEndian.Uint32(mainBlock[1:5])
}

func (o *Output) Amount() uint64 {
	mainBlock := o.blocks.At(int(OutputBlockMain))
	return binary.BigEndian.Uint64(mainBlock[5:])
}

func (o *Output) NumBlocks() int {
	return o.blocks.NumElements()
}

func (o *Output) ForEachBlock(fun func(idx byte, blockData []byte) bool) {
	o.blocks.ForEach(func(i int, data []byte) bool {
		return fun(byte(i), data)
	})
}
