package ledger

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
)

func TimeLockConstraint(ts uint32) Constraint {
	src := fmt.Sprintf("timeLock(u32/%d)", ts)
	_, _, binCode, err := easyfl.CompileExpression(src)
	easyfl.AssertNoError(err)
	return binCode
}

var (
	timeLockConstraintPrefix []byte
	timeLockConstraintLen    int
)

func initTimeLockConstraint() {
	prefix, err := easyfl.FunctionCallPrefixByName("timeLock", 1)
	easyfl.AssertNoError(err)
	common.Assert(0 < len(prefix) && len(prefix) <= 2, "0<len(prefix) && len(prefix)<=2")
	template := TimeLockConstraint(0)
	timeLockConstraintLen = len(template)
	lenConstraintPrefix := len(prefix) + 1
	common.Assert(len(template) == len(prefix)+1+8, "len(template)==len(prefix)+1+8")
	timeLockConstraintPrefix = easyfl.Concat(template[:lenConstraintPrefix])
}

// TimeLockFromConstraint extracts sender address ($0) from the sender script
func TimeLockFromConstraint(data []byte) (uint32, error) {
	if !bytes.HasPrefix(data, timeLockConstraintPrefix) {
		return 0, fmt.Errorf("TimeLockFromConstraint:: not a sender constraint")
	}
	if len(data) < len(timeLockConstraintPrefix)+1 {
		return 0, fmt.Errorf("TimeLockFromConstraint:: wrong data len")
	}
	timeLockBytes, success, err := easyfl.ParseInlineDataPrefix(data[len(senderConstraintPrefix):])
	if err != nil || !success || len(timeLockBytes) != 4 {
		return 0, fmt.Errorf("failed to parse time lock bytes")
	}
	return binary.BigEndian.Uint32(timeLockBytes), nil
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
