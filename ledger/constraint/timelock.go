package constraint

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
)

func Timelock(ts uint32) []byte {
	src := fmt.Sprintf("timelock(u32/%d)", ts)
	_, _, binCode, err := easyfl.CompileExpression(src)
	easyfl.AssertNoError(err)
	return binCode
}

func initTimelockConstraint() {
	easyfl.MustExtendMany(TimeLockConstraintSource)

	example := Timelock(1337)
	prefix, args, err := easyfl.ParseCallWithConstants(example, 1)
	easyfl.AssertNoError(err)
	common.Assert(len(args[0]) == 4 && binary.BigEndian.Uint32(args[0]) == 1337, "len(args[0]) == 4 && binary.BigEndian.Uint32(args[0]) == 1337")
	registerConstraint("timelock", prefix)
}

// TimelockFromConstraint extracts sender address ($0) from the sender script
func TimelockFromConstraint(data []byte) (uint32, error) {
	prefix, args, err := easyfl.ParseCallWithConstants(data, 1)
	if err != nil {
		return 0, err
	}
	prefix1, ok := PrefixByName("timelock")
	common.Assert(ok, "no timelock")
	if !bytes.Equal(prefix, prefix1) {
		return 0, fmt.Errorf("TimelockFromConstraint:: not a 'timelock' constraint")
	}
	if len(args[0]) != 4 {
		return 0, fmt.Errorf("TimelockFromConstraint:: wrong data length")
	}
	return binary.BigEndian.Uint32(args[0]), nil
}

const TimeLockConstraintSource = `

// enforces output can be unlocked only after specified time
// $0 is Unix seconds of the time lock
func timelock: or(
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
