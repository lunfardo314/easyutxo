package txbuilder

import (
	"fmt"
	"sort"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraint"
)

type Output struct {
	arr *lazyslice.Array
}

type OutputWithID struct {
	ID     ledger.OutputID
	Output *Output
}

func NewOutput() *Output {
	ret := &Output{
		arr: lazyslice.EmptyArray(256),
	}

	return ret
}

func OutputBasic(amount uint64, ts uint32, lock constraint.Lock) *Output {
	return NewOutput().WithAmount(amount).WithTimestamp(ts).WithLockConstraint(lock)
}

func OutputFromBytes(data []byte) (*Output, error) {
	ret := &Output{
		arr: lazyslice.ArrayFromBytes(data, 256),
	}
	if ret.arr.NumElements() < 3 {
		return nil, fmt.Errorf("at least 3 constraints expected")
	}
	if _, err := constraint.AmountFromBytes(ret.arr.At(int(ledger.OutputBlockAmount))); err != nil {
		return nil, err
	}
	if _, err := constraint.TimestampFromBytes(ret.arr.At(int(ledger.OutputBlockTimestamp))); err != nil {
		return nil, err
	}
	if _, err := constraint.LockFromBytes(ret.arr.At(int(ledger.OutputBlockLock))); err != nil {
		return nil, err
	}
	return ret, nil
}

func (o *Output) WithAmount(amount uint64) *Output {
	o.arr.PutAtIdxGrow(ledger.OutputBlockAmount, constraint.NewAmount(amount).Bytes())
	return o
}

func (o *Output) WithTimestamp(ts uint32) *Output {
	o.arr.PutAtIdxGrow(ledger.OutputBlockTimestamp, constraint.NewTimestamp(ts).Bytes())
	return o
}

func (o *Output) Amount() uint64 {
	ret, err := constraint.AmountFromBytes(o.arr.At(int(ledger.OutputBlockAmount)))
	easyfl.AssertNoError(err)
	return uint64(ret)
}

func (o *Output) Timestamp() uint32 {
	ret, err := constraint.TimestampFromBytes(o.arr.At(int(ledger.OutputBlockTimestamp)))
	easyfl.AssertNoError(err)
	return uint32(ret)
}

func (o *Output) WithLockConstraint(lock constraint.Lock) *Output {
	o.PutConstraint(lock.Bytes(), ledger.OutputBlockLock)
	return o
}

func (o *Output) AsArray() *lazyslice.Array {
	return o.arr
}

func (o *Output) Bytes() []byte {
	return o.arr.Bytes()
}

func (o *Output) PushConstraint(c []byte) (byte, error) {
	if o.NumConstraints() >= 256 {
		return 0, fmt.Errorf("too many constraints")
	}
	o.arr.Push(c)
	return byte(o.arr.NumElements() - 1), nil
}

func (o *Output) PutConstraint(c []byte, idx byte) {
	o.arr.PutAtIdxGrow(idx, c)
}

func (o *Output) Constraint(idx byte) []byte {
	return o.arr.At(int(idx))
}

func (o *Output) NumConstraints() int {
	return o.arr.NumElements()
}

func (o *Output) ForEachConstraint(fun func(idx byte, constr []byte) bool) {
	o.arr.ForEach(func(i int, data []byte) bool {
		return fun(byte(i), data)
	})
}

// Sender looks for sender constraint in the output and parses out sender address
//func (o *Output) Sender() ([]byte, bool) {
//	var ret []byte
//	var found bool
//	o.ForEachConstraint(func(idx byte, constr []byte) bool {
//		if idx == OutputBlockAmount || idx == OutputBlockTimestamp || idx == OutputBlockLock {
//			return true
//		}
//		ret, found = constraint.SenderFromConstraint(constr)
//		if found {
//			return false
//		}
//		return true
//	})
//	if found {
//		return ret, true
//	}
//	return nil, false
//}

func (o *Output) Lock() constraint.Lock {
	ret, err := constraint.LockFromBytes(o.arr.At(int(ledger.OutputBlockLock)))
	easyfl.AssertNoError(err)
	return ret
}

func (o *Output) TimeLock() (uint32, bool) {
	var ret constraint.Timelock
	var err error
	found := false
	o.ForEachConstraint(func(idx byte, constr []byte) bool {
		if idx == ledger.OutputBlockAmount || idx == ledger.OutputBlockTimestamp || idx == ledger.OutputBlockLock {
			return true
		}
		ret, err = constraint.TimelockFromBytes(constr)
		if err == nil {
			// TODO optimize parsing
			found = true
			return false
		}
		return true
	})
	if found {
		return uint32(ret), true
	}
	return 0, false
}

func ParseAndSortOutputData(outs []*ledger.OutputDataWithID, desc ...bool) ([]*OutputWithID, error) {
	ret := make([]*OutputWithID, len(outs))
	for i, od := range outs {
		out, err := OutputFromBytes(od.OutputData)
		if err != nil {
			return nil, err
		}
		ret[i] = &OutputWithID{
			ID:     od.ID,
			Output: out,
		}
	}
	if len(desc) > 0 && desc[0] {
		sort.Slice(ret, func(i, j int) bool {
			return ret[i].Output.Amount() > ret[j].Output.Amount()
		})
	} else {
		sort.Slice(ret, func(i, j int) bool {
			return ret[i].Output.Amount() < ret[j].Output.Amount()
		})
	}
	return ret, nil
}
