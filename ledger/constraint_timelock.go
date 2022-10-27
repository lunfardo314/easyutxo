package ledger

import (
	"fmt"

	"github.com/lunfardo314/easyfl"
)

func TimeLockConstraint(ts uint32) Constraint {
	src := fmt.Sprintf("timeLock(u32/%d)", ts)
	_, _, binCode, err := easyfl.CompileExpression(src)
	easyfl.AssertNoError(err)
	return binCode
}

const TimeLockConstraintSource = `

// enforces output can be unlocked only after specified time
// $0 is Unix seconds of the time lock
func timeLock: or(
	and( 
		isProducedBranch(@), 
		equal(len8($0), 4),             // must be 4-bytes long
		lessThan(txTimestampBytes, $0)  // time lock must be after the transaction (not very necessary)
	), 
	and( 
		isConsumedBranch(@), 
		lessThan($0, txTimestampBytes)  // is unlocked if tx timestamp is strongly after the time lock 
	) 
)
`
