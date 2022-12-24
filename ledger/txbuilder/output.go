package txbuilder

import (
	"fmt"
	"sort"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraints"
)

type Output struct {
	arr *lazyslice.Array
}

type OutputWithID struct {
	ID     ledger.OutputID
	Output *Output
}

type OutputWithChainID struct {
	OutputWithID
	ChainID                    [32]byte
	PredecessorConstraintIndex byte
}

func NewOutput() *Output {
	ret := &Output{
		arr: lazyslice.EmptyArray(256),
	}

	return ret
}

func OutputBasic(amount uint64, ts uint32, lock constraints.Lock) *Output {
	return NewOutput().WithAmount(amount).WithTimestamp(ts).WithLock(lock)
}

func OutputFromBytes(data []byte) (*Output, error) {
	ret := &Output{
		arr: lazyslice.ArrayFromBytes(data, 256),
	}
	if ret.arr.NumElements() < 3 {
		return nil, fmt.Errorf("at least 3 constraints expected")
	}
	if _, err := constraints.AmountFromBytes(ret.arr.At(int(constraints.ConstraintIndexAmount))); err != nil {
		return nil, err
	}
	if _, err := constraints.TimestampFromBytes(ret.arr.At(int(constraints.ConstraintIndexTimestamp))); err != nil {
		return nil, err
	}
	if _, err := constraints.LockFromBytes(ret.arr.At(int(constraints.ConstraintIndexLock))); err != nil {
		return nil, err
	}
	return ret, nil
}

func (o *Output) WithAmount(amount uint64) *Output {
	o.arr.PutAtIdxGrow(constraints.ConstraintIndexAmount, constraints.NewAmount(amount).Bytes())
	return o
}

func (o *Output) WithTimestamp(ts uint32) *Output {
	o.arr.PutAtIdxGrow(constraints.ConstraintIndexTimestamp, constraints.NewTimestamp(ts).Bytes())
	return o
}

func (o *Output) Amount() uint64 {
	ret, err := constraints.AmountFromBytes(o.arr.At(int(constraints.ConstraintIndexAmount)))
	easyfl.AssertNoError(err)
	return uint64(ret)
}

func (o *Output) Timestamp() uint32 {
	ret, err := constraints.TimestampFromBytes(o.arr.At(int(constraints.ConstraintIndexTimestamp)))
	easyfl.AssertNoError(err)
	return uint32(ret)
}

func (o *Output) WithLock(lock constraints.Lock) *Output {
	o.PutConstraint(lock.Bytes(), constraints.ConstraintIndexLock)
	return o
}

func (o *Output) AsArray() *lazyslice.Array {
	return o.arr
}

func (o *Output) Bytes() []byte {
	return o.arr.Bytes()
}

func (o *Output) Clone() *Output {
	ret, err := OutputFromBytes(o.Bytes())
	easyfl.AssertNoError(err)
	return ret
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
//		if idx == ConstraintIndexAmount || idx == ConstraintIndexTimestamp || idx == ConstraintIndexLock {
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

func (o *Output) Lock() constraints.Lock {
	ret, err := constraints.LockFromBytes(o.arr.At(int(constraints.ConstraintIndexLock)))
	easyfl.AssertNoError(err)
	return ret
}

func (o *Output) TimeLock() (uint32, bool) {
	var ret constraints.Timelock
	var err error
	found := false
	o.ForEachConstraint(func(idx byte, constr []byte) bool {
		if idx == constraints.ConstraintIndexAmount || idx == constraints.ConstraintIndexTimestamp || idx == constraints.ConstraintIndexLock {
			return true
		}
		ret, err = constraints.TimelockFromBytes(constr)
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

func (o *Output) SenderAddressED25519() (constraints.AddressED25519, bool) {
	var ret *constraints.SenderAddressED25519
	var err error
	found := false
	o.ForEachConstraint(func(idx byte, constr []byte) bool {
		if idx == constraints.ConstraintIndexAmount || idx == constraints.ConstraintIndexTimestamp || idx == constraints.ConstraintIndexLock {
			return true
		}
		ret, err = constraints.SenderAddressED25519FromBytes(constr)
		if err == nil {
			found = true
			return false
		}
		return true
	})
	if found {
		return ret.Address, true
	}
	return nil, false
}

// ChainConstraint finds and parses chain constraint. Returns its constraintIndex or 0xff if not found
func (o *Output) ChainConstraint() (*constraints.ChainConstraint, byte) {
	var ret *constraints.ChainConstraint
	var err error
	found := byte(0xff)
	o.ForEachConstraint(func(idx byte, constr []byte) bool {
		if idx == constraints.ConstraintIndexAmount || idx == constraints.ConstraintIndexTimestamp || idx == constraints.ConstraintIndexLock {
			return true
		}
		ret, err = constraints.ChainConstraintFromBytes(constr)
		if err == nil {
			found = idx
			return false
		}
		return true
	})
	if found != 0xff {
		return ret, found
	}
	return nil, 0xff
}

func ParseAndSortOutputData(outs []*ledger.OutputDataWithID, filter func(o *Output) bool, desc ...bool) ([]*OutputWithID, error) {
	ret := make([]*OutputWithID, 0, len(outs))
	for _, od := range outs {
		out, err := OutputFromBytes(od.OutputData)
		if err != nil {
			return nil, err
		}
		if filter != nil && !filter(out) {
			continue
		}
		ret = append(ret, &OutputWithID{
			ID:     od.ID,
			Output: out,
		})
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

func (o *Output) ToString(prefix ...string) string {
	ret := ""
	pref := ""
	if len(prefix) > 0 {
		pref = prefix[0]
	}
	o.arr.ForEach(func(i int, data []byte) bool {
		c, err := constraints.FromBytes(data)
		if err != nil {
			ret += fmt.Sprintf("%s%d: %v (%d bytes)\n", pref, i, err, len(data))
		} else {
			ret += fmt.Sprintf("%s%d: %s (%d bytes)\n", pref, i, c.String(), len(data))
		}
		return true
	})
	return ret
}
