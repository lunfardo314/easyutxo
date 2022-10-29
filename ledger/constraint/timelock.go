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
const (
	timelockName     = "timelock"
	timelockTemplate = timelockName + "(u32/%d)"
)

type Timelock uint32

func NewTimelock(unixSec uint32) Timelock {
	return Timelock(unixSec)
}

func (t Timelock) Name() string {
	return timelockName
}

func (t Timelock) Bytes() []byte {
	return mustBinFromSource(t.source())
}

func (t Timelock) String() string {
	return fmt.Sprintf("%s(%d)", timelockName, uint32(t))
}

func (t Timelock) source() string {
	return fmt.Sprintf(timelockTemplate, t)
}

func TimelockFromBytes(data []byte) (Timelock, error) {
	sym, _, args, err := easyfl.ParseBinaryOneLevel(data, 1)
	if err != nil {
		return 0, err
	}
	if sym != timelockName {
		return 0, fmt.Errorf("not a timelock constraint")
	}
	tlBin := easyfl.StripDataPrefix(args[0])
	common.Assert(len(tlBin) == 4, "can't parse timelock")
	return Timelock(binary.BigEndian.Uint32(tlBin)), nil
}

func initTimelockConstraint() {
	easyfl.MustExtendMany(timelockSource)

	example := NewTimelock(1337)
	sym, prefix, args, err := easyfl.ParseBinaryOneLevel(example.Bytes(), 1)
	easyfl.AssertNoError(err)
	tlBin := easyfl.StripDataPrefix(args[0])
	common.Assert(sym == timelockName && len(tlBin) == 4 && binary.BigEndian.Uint32(tlBin) == 1337, "inconsistency in 'timelock'")

	registerConstraint(timelockName, prefix, func(data []byte) (Constraint, error) {
		return TimestampFromBytes(data)
	})
}
