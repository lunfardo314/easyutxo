package ledger

import (
	"errors"
	"fmt"

	"github.com/lunfardo314/easyfl"
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

const (
	OutputBlockAmount = byte(iota)
	OutputBlockTimestamp
	OutputBlockLock
	OutputNumMandatoryBlocks
)

type (
	OutputID [OutputIDLength]byte

	Output struct {
		arr *lazyslice.Array
	}

	Constraint []byte

	OutputWithID struct {
		ID     OutputID
		Output *Output
	}
)

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

func (oid *OutputID) String() string {
	txid := oid.TransactionID()
	return fmt.Sprintf("[%d]%s", oid.Index(), txid.String())
}

func (oid *OutputID) TransactionID() (ret TransactionID) {
	copy(ret[:], oid[:TransactionIDLength])
	return
}

func (oid *OutputID) Index() byte {
	return oid[TransactionIDLength]
}

func (oid *OutputID) Bytes() []byte {
	return oid[:]
}

func NewOutput() *Output {
	ret := &Output{
		arr: lazyslice.EmptyArray(256),
	}

	return ret
}

func OutputFromBytes(data []byte) (*Output, error) {
	ret := &Output{
		arr: lazyslice.ArrayFromBytes(data, 256),
	}
	if ret.arr.NumElements() < 3 {
		return nil, fmt.Errorf("at least 3 constraints expected")
	}
	var err error
	if _, err = AmountFromConstraint(ret.arr.At(int(OutputBlockAmount))); err != nil {
		return nil, err
	}
	if _, err = TimestampFromConstraint(ret.arr.At(int(OutputBlockTimestamp))); err != nil {
		return nil, err
	}

	if _, err = LockFromBytes(ret.arr.At(int(OutputBlockLock))); err != nil {
		return nil, err
	}
	return ret, nil
}

func (o *Output) SetAmount(amount uint64) {
	o.arr.PutAtIdxGrow(OutputBlockAmount, AmountConstraint(amount))
}

func (o *Output) SetTimestamp(ts uint32) {
	o.arr.PutAtIdxGrow(OutputBlockTimestamp, TimestampConstraint(ts))
}

func (o *Output) Amount() uint64 {
	ret, err := AmountFromConstraint(o.arr.At(int(OutputBlockAmount)))
	easyfl.AssertNoError(err)
	return ret
}

func (o *Output) Timestamp() uint32 {
	ret, err := TimestampFromConstraint(o.arr.At(int(OutputBlockTimestamp)))
	easyfl.AssertNoError(err)
	return ret
}

func (o *Output) PutLockConstraint(data []byte) {
	o.PutConstraint(data, OutputBlockLock)
}

func (o *Output) AsArray() *lazyslice.Array {
	return o.arr
}

func (o *Output) Bytes() []byte {
	return o.arr.Bytes()
}

func (o *Output) PushConstraint(c Constraint) (byte, error) {
	if o.NumConstraints() >= 256 {
		return 0, fmt.Errorf("too many constraints")
	}
	o.arr.Push(c)
	return byte(o.arr.NumElements() - 1), nil
}

func (o *Output) PutConstraint(c Constraint, idx byte) {
	o.arr.PutAtIdxGrow(idx, c)
}

func (o *Output) Constraint(idx byte) Constraint {
	return o.arr.At(int(idx))
}

func (o *Output) NumConstraints() int {
	return o.arr.NumElements()
}

func (o *Output) ForEachConstraint(fun func(idx byte, constraint Constraint) bool) {
	o.arr.ForEach(func(i int, data []byte) bool {
		return fun(byte(i), Constraint(data))
	})
}

func (o *Output) Sender() (Lock, bool) {
	panic("not implemented")
	//var ret Lock
	//found := false
	//o.ForEachConstraint(func(idx byte, constraint Constraint) bool {
	//	if constraint.Type() == ConstraintTypeSender {
	//		ret = Sender(constraint).Lock()
	//		found = true
	//		return false
	//	}
	//	return true
	//})
	//return ret, found
}

func (o *Output) TimeLock() (uint32, bool) {
	panic("not implemented")
	//var ret uint32
	//found := false
	//o.ForEachConstraint(func(idx byte, constraint Constraint) bool {
	//	if constraint.Type() == ConstraintTypeTimeLock {
	//		ret = binary.BigEndian.Uint32(constraint.Data())
	//		found = true
	//		return false
	//	}
	//	return true
	//})
	//return ret, found
}

//---------------------------------------------------------

func (u *UnlockParams) Bytes() []byte {
	return u.array.Bytes()
}

func NewUnlockBlock() *UnlockParams {
	return &UnlockParams{
		array: lazyslice.EmptyArray(256),
	}
}

// PutUnlockParams puts data at index. If index is out of bounds, pushes empty elements to fill the gaps
// Unlock params in the element 'idx' corresponds to the consumed output constraint at the same index
func (u *UnlockParams) PutUnlockParams(data []byte, idx byte) {
	u.array.PutAtIdxGrow(idx, data)
}
