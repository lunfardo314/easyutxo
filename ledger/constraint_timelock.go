package ledger

import (
	"encoding/binary"

	"github.com/lunfardo314/easyfl"
)

func NewTimeLockConstraint(ts uint32) Constraint {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], ts)
	return easyfl.Concat(byte(ConstraintTypeTimeLock), b[:])
}

const TimeLockConstraintSource = `

// enforces output can be unlocked only after specified time
func timeLock: or(
	and( isProducedBranch(@), equal(len8(selfConstraintData), 4) ), // must be 4 bytes long
	and( isConsumedBranch(@), lessThan(selfConstraintData, txTimestampBytes) ) // transaction must be strongly after timelock
)
`
