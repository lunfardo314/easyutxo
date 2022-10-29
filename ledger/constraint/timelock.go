package constraint

import (
	"encoding/binary"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
)

const timelockSource = `
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

func TimelockConstraintSource(ts uint32) string {
	return fmt.Sprintf("timelock(u32/%d)", ts)
}

func TimelockConstraintBin(ts uint32) []byte {
	return mustBinFromSource(TimelockConstraintSource(ts))
}

func initTimelockConstraint() {
	easyfl.MustExtendMany(timelockSource)

	example := TimelockConstraintBin(1337)
	sym, prefix, args, err := easyfl.ParseBinaryOneLevel(example, 1)
	easyfl.AssertNoError(err)
	tlBin := easyfl.StripDataPrefix(args[0])
	common.Assert(sym == "timelock" && len(tlBin) == 4 && binary.BigEndian.Uint32(tlBin) == 1337, "inconsistency in 'timelock'")
	registerConstraint("timelock", prefix)
}

// TimelockFromBin extracts timelock ($0) from the timelock script
func TimelockFromBin(data []byte) (uint32, bool) {
	sym, _, args, err := easyfl.ParseBinaryOneLevel(data, 1)
	if err != nil {
		return 0, false
	}
	if sym != "timelock" {
		return 0, false
	}
	tlBin := easyfl.StripDataPrefix(args[0])
	common.Assert(len(tlBin) == 4, "inconsistency in 'timelock'")
	return binary.BigEndian.Uint32(tlBin), true
}
