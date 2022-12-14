package constraints

import (
	"encoding/binary"
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/unitrie/common"
)

const timestampSource = `
// $0 - 4 bytes Unix seconds big-endian 
func timestamp: and(
	equal(selfBlockIndex,1),  // must be at block 1
	or(
		// for 'produced' output $0 must be equal to the transaction timestamp 
		and( selfIsProducedOutput, equal($0, txTimestampBytes) ),
		// for 'consumed' output $0 must be strictly before the transaction timestamp 
		and( selfIsConsumedOutput, lessThan($0, txTimestampBytes) )	
	)
)
`

const (
	timestampName     = "timestamp"
	timestampTemplate = timestampName + "(u32/%d)"
)

type Timestamp uint32

func (t Timestamp) Name() string {
	return timestampName
}

func (t Timestamp) source() string {
	return fmt.Sprintf("timestamp(u32/%d)", uint32(t))
}

func (t Timestamp) Bytes() []byte {
	return mustBinFromSource(t.source())
}

func (t Timestamp) String() string {
	return fmt.Sprintf(timestampTemplate, uint32(t))
}

func NewTimestamp(unixSec uint32) Timestamp {
	return Timestamp(unixSec)
}

func TimestampFromBytes(data []byte) (Timestamp, error) {
	sym, _, args, err := easyfl.ParseBytecodeOneLevel(data, 1)
	if err != nil {
		return 0, err
	}
	tsBin := easyfl.StripDataPrefix(args[0])
	if sym != timestampName || len(tsBin) != 4 {
		return 0, fmt.Errorf("can't parse timestamp constraint")
	}
	return Timestamp(binary.BigEndian.Uint32(tsBin)), nil
}

func initTimestampConstraint() {
	easyfl.MustExtendMany(timestampSource)

	example := NewTimestamp(1337)
	sym, prefix, args, err := easyfl.ParseBytecodeOneLevel(example.Bytes(), 1)
	easyfl.AssertNoError(err)
	tsBin := easyfl.StripDataPrefix(args[0])
	common.Assert(sym == timestampName && len(tsBin) == 4 && binary.BigEndian.Uint32(tsBin) == 1337, "'timestamp' consistency check failed")

	registerConstraint(timestampName, prefix, func(data []byte) (Constraint, error) {
		return TimestampFromBytes(data)
	})
}
