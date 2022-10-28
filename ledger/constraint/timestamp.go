package constraint

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
)

const timestampSource = `
// $0 - 4 bytes Unix seconds big-endian 
func timestamp: or(
	and( isProducedBranch(@), equal($0, txTimestampBytes) ),
	and( isConsumedBranch(@), lessThan($0, txTimestampBytes) )	
)
`

func TimestampConstraintSource(unixSec uint32) string {
	return fmt.Sprintf("timestamp(u32/%d)", unixSec)
}

func TimestampConstraintBin(unixSec uint32) []byte {
	return mustBinFromSource(TimestampConstraintSource(unixSec))
}

func initTimestampConstraint() {
	easyfl.MustExtendMany(timestampSource)

	example := TimestampConstraintBin(1337)
	prefix, args, err := easyfl.ParseCallWithConstants(example, 1)
	easyfl.AssertNoError(err)
	common.Assert(len(args[0]) == 4 && binary.BigEndian.Uint32(args[0]) == 1337, "len(args[0]) == 4 && binary.BigEndian.Uint32(args[0]) == 1337")
	registerConstraint("timestamp", prefix)
}

func TimestampFromConstraint(data []byte) (uint32, error) {
	prefix, args, err := easyfl.ParseCallWithConstants(data, 1)
	if err != nil {
		return 0, err
	}
	prefix1, ok := PrefixByName("timestamp")
	common.Assert(ok, "no timestamp")
	if !bytes.Equal(prefix, prefix1) {
		return 0, fmt.Errorf("TimestampFromConstraint:: not an 'amount' constraint")
	}
	if len(args[0]) != 4 {
		return 0, fmt.Errorf("TimestampFromConstraint:: wrong data length")
	}
	return binary.BigEndian.Uint32(args[0]), nil
}
