package ledger

import (
	"errors"
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger/constraint"
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

func OutputBasic(amount uint64, ts uint32, addr []byte) *Output {
	return NewOutput().WithAmount(amount).WithTimestamp(ts).WithLockConstraint(addr)
}

func OutputFromBytes(data []byte) (*Output, error) {
	ret := &Output{
		arr: lazyslice.ArrayFromBytes(data, 256),
	}
	if ret.arr.NumElements() < 3 {
		return nil, fmt.Errorf("at least 3 constraints expected")
	}
	var err error
	if _, err = constraint.AmountFromConstraint(ret.arr.At(int(OutputBlockAmount))); err != nil {
		return nil, err
	}
	if _, err = constraint.TimestampFromConstraint(ret.arr.At(int(OutputBlockTimestamp))); err != nil {
		return nil, err
	}

	lock := ret.arr.At(int(OutputBlockLock))
	if !constraint.IsKnownLock(lock) {
		return nil, fmt.Errorf("wrong or unsupported type of lock: %s", easyfl.Fmt(lock))
	}
	return ret, nil
}

func (o *Output) WithAmount(amount uint64) *Output {
	o.arr.PutAtIdxGrow(OutputBlockAmount, constraint.Amount(amount))
	return o
}

func (o *Output) WithTimestamp(ts uint32) *Output {
	o.arr.PutAtIdxGrow(OutputBlockTimestamp, constraint.Timestamp(ts))
	return o
}

func (o *Output) Amount() uint64 {
	ret, err := constraint.AmountFromConstraint(o.arr.At(int(OutputBlockAmount)))
	easyfl.AssertNoError(err)
	return ret
}

func (o *Output) Timestamp() uint32 {
	ret, err := constraint.TimestampFromConstraint(o.arr.At(int(OutputBlockTimestamp)))
	easyfl.AssertNoError(err)
	return ret
}

func (o *Output) WithLockConstraint(data []byte) *Output {
	o.PutConstraint(data, OutputBlockLock)
	return o
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

// Sender looks for sender constraint in the output and parses out sender address
func (o *Output) Sender() ([]byte, bool) {
	var ret []byte
	var found bool
	o.ForEachConstraint(func(idx byte, constr Constraint) bool {
		if idx == OutputBlockAmount || idx == OutputBlockTimestamp || idx == OutputBlockLock {
			return true
		}
		var err error
		if ret, err = constraint.SenderFromConstraint(constr); err == nil {
			found = true
			return false
		}
		return false
	})
	if found {
		return ret, true
	}
	return nil, false
}

func (o *Output) Lock() []byte {
	return o.arr.At(int(OutputBlockLock))
}

func (o *Output) TimeLock() (uint32, bool) {
	var ret uint32
	var found bool
	o.ForEachConstraint(func(idx byte, constr Constraint) bool {
		if idx == OutputBlockAmount || idx == OutputBlockTimestamp || idx == OutputBlockLock {
			return true
		}
		var err error
		if ret, err = constraint.TimelockFromConstraint(constr); err == nil {
			found = true
			return false
		}
		return false
	})
	if found {
		return ret, true
	}
	return 0, false
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
