package ledger

import (
	"encoding/binary"
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
	OutputBlockMain = byte(iota)
	OutputBlockLock
	OutputNumMandatoryBlocks
)

type (
	OutputID [OutputIDLength]byte

	Output struct {
		// Mandatory constraints are:
		// - #0 amount/timestamp - main constraint
		// - #1 address constraint for indexing. May be nil, then it is not indexed
		// - #2 sender constraint
		Amount    uint64
		Timestamp uint32
		Lock      Lock
		// other constraints are optional
		OptionalConstraints []Constraint
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

func NewOutput(amount uint64, timestamp uint32, address Lock) *Output {
	ret := &Output{
		Amount:              amount,
		Timestamp:           timestamp,
		Lock:                address,
		OptionalConstraints: make([]Constraint, 0),
	}
	return ret
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

func OutputFromBytes(data []byte) (*Output, error) {
	arr := lazyslice.ArrayFromBytes(data, 256)
	if arr.NumElements() < 2 {
		return nil, fmt.Errorf("wrong data length")
	}
	ret := NewOutput(0, 0, nil)
	err := ret.parseMainConstraint(arr.At(int(OutputBlockMain)))
	if err != nil {
		return nil, err
	}
	ret.Lock, err = LockFromBytes(arr.At(int(OutputBlockLock)))
	if err != nil {
		return nil, err
	}
	for i := int(OutputNumMandatoryBlocks); i < arr.NumElements(); i++ {
		d := arr.At(i)
		if len(d) < 1 {
			return nil, fmt.Errorf("wrong data length")
		}
		ret.OptionalConstraints = append(ret.OptionalConstraints, arr.At(i))
	}
	return ret, nil
}

func (o *Output) ToArray() *lazyslice.Array {
	ret := lazyslice.EmptyArray(256)
	var a [8]byte
	binary.BigEndian.PutUint64(a[:], o.Amount)
	ret.Push(o.mainConstraint()) // @ OutputBlockMain = 0
	ret.Push(o.Lock)             // @ OutputBlockLock = 1
	for _, c := range o.OptionalConstraints {
		ret.Push(c)
	}
	return ret
}

func (o *Output) Bytes() []byte {
	return o.ToArray().Bytes()
}

func (o *Output) PushConstraint(c Constraint) (byte, error) {
	if o.NumConstraints() >= 256 {
		return 0, fmt.Errorf("too many constraints")
	}
	o.OptionalConstraints = append(o.OptionalConstraints, c)
	return byte(len(o.OptionalConstraints) + 1), nil
}

func (o *Output) Constraint(idx byte) Constraint {
	switch idx {
	case 0:
		return o.mainConstraint()
	case 1:
		return Constraint(o.Lock)
	default:
		return o.OptionalConstraints[idx]
	}
}

const mainConstraintSize = 1 + 4 + 8

func (o *Output) mainConstraint() Constraint {
	var a [8]byte
	var ts [4]byte
	binary.BigEndian.PutUint64(a[:], o.Amount)
	binary.BigEndian.PutUint32(ts[:], o.Timestamp)

	ret := easyfl.Concat(byte(ConstraintTypeMain), ts[:], a[:])
	easyfl.Assert(len(ret) == mainConstraintSize, "len(ret)==mainConstraintSize")
	return ret
}

func (o *Output) parseMainConstraint(data []byte) error {
	if len(data) != 1+4+8 {
		return fmt.Errorf("wrong data length")
	}
	if ConstraintType(data[0]) != ConstraintTypeMain {
		return fmt.Errorf("main constraint code expected")
	}
	o.Timestamp = binary.BigEndian.Uint32(data[1:5])
	o.Amount = binary.BigEndian.Uint64(data[5:])
	return nil
}

func (c Constraint) Type() ConstraintType {
	return ConstraintType(c[0])
}

func (c Constraint) Name() string {
	_, name := mustGetConstraintBinary(ConstraintType(c[0]))
	return name
}

func (o *Output) NumConstraints() int {
	return len(o.OptionalConstraints) + 2
}

func (o *Output) ForEachConstraint(fun func(idx byte, constraint Constraint) bool) {
	if !fun(OutputBlockMain, o.mainConstraint()) {
		return
	}
	if !fun(OutputBlockLock, Constraint(o.Lock)) {
		return
	}
	for idx, c := range o.OptionalConstraints {
		if !fun(byte(idx+int(OutputNumMandatoryBlocks)), c) {
			return
		}
	}
}

func (o *Output) Sender() Sender {
	var ret Sender
	o.ForEachConstraint(func(idx byte, constraint Constraint) bool {
		if constraint.Type() == ConstraintTypeSender {
			ret = Sender(constraint)
			return false
		}
		return true
	})
	return ret
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
	for u.array.NumElements() <= int(idx) {
		u.array.Push(nil)
	}
	u.array.PutAtIdx(idx, data)
}
