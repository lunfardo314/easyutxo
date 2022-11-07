package library

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/lunfardo314/easyfl"
)

type DeadlineLock struct {
	Deadline         uint32
	ConstraintMain   Accountable
	ConstraintExpiry Accountable
}

const (
	deadlineLockName     = "deadlineLock"
	deadlineLockTemplate = deadlineLockName + "(u32/%d,x/%s, x/%s)"
)

func NewDeadlineLock(deadline uint32, main, expiry Accountable) *DeadlineLock {
	return &DeadlineLock{
		Deadline:         deadline,
		ConstraintMain:   main,
		ConstraintExpiry: expiry,
	}
}

func (dl *DeadlineLock) source() string {
	return fmt.Sprintf(deadlineLockTemplate,
		dl.Deadline,
		hex.EncodeToString(dl.ConstraintMain.AccountID()),
		hex.EncodeToString(dl.ConstraintExpiry.AccountID()),
	)
}

func (dl *DeadlineLock) Bytes() []byte {
	return mustBinFromSource(dl.source())
}

func (dl *DeadlineLock) String() string {
	return fmt.Sprintf("%s(%d,%s,%s)", deadlineLockName, dl.Deadline, dl.ConstraintMain, dl.ConstraintExpiry)
}

func (dl *DeadlineLock) IndexableTags() []Accountable {
	return []Accountable{dl.ConstraintMain, dl.ConstraintExpiry}
}

func (dl *DeadlineLock) UnlockableWith(acc AccountID, ts uint32) bool {
	if ts <= dl.Deadline {
		return bytes.Equal(dl.ConstraintMain.AccountID(), acc)
	}
	return bytes.Equal(dl.ConstraintExpiry.AccountID(), acc)
}

func (dl *DeadlineLock) Name() string {
	return deadlineLockName
}

func initDeadlineLockConstraint() {
	easyfl.MustExtendMany(deadlineLockSource)

	example := NewDeadlineLock(1337, AddressED25519Null(), AddressED25519Null())
	lockBack, err := DeadlineLockFromBytes(example.Bytes())
	easyfl.AssertNoError(err)

	easyfl.Assert(Equal(lockBack.ConstraintMain, AddressED25519Null()), "inconsistency "+deadlineLockName)
	easyfl.Assert(Equal(lockBack.ConstraintExpiry, AddressED25519Null()), "inconsistency "+deadlineLockName)

	prefix, err := easyfl.ParseCallPrefixFromBinary(example.Bytes())
	easyfl.AssertNoError(err)

	registerConstraint(deadlineLockName, prefix, func(data []byte) (Constraint, error) {
		return DeadlineLockFromBytes(data)
	})
}

func DeadlineLockFromBytes(data []byte) (*DeadlineLock, error) {
	sym, _, args, err := easyfl.ParseBinaryOneLevel(data, 3)
	if err != nil {
		return nil, err
	}
	ret := &DeadlineLock{}
	dlBin := easyfl.StripDataPrefix(args[0])
	if sym != deadlineLockName || len(dlBin) != 4 {
		return nil, fmt.Errorf("can't parse deadline lock")
	}
	ret.Deadline = binary.BigEndian.Uint32(dlBin)

	if ret.ConstraintMain, err = AccountableFromBytes(args[1]); err != nil {
		return nil, err
	}
	if ret.ConstraintExpiry, err = AccountableFromBytes(args[2]); err != nil {
		return nil, err
	}
	return ret, nil
}

const deadlineLockSource = `

func deadlineLock: and(
	equal(selfBlockIndex,2), // locks must be at block 2
	if(
		lessThan($0, txTimestampBytes),
		$1, 
		$2
	)
)
`
