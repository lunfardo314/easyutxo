package constraint

import (
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
	sym, prefix, args, err := easyfl.ParseBinaryOneLevel(example, 1)
	easyfl.AssertNoError(err)
	tsBin := easyfl.StripDataPrefix(args[0])
	common.Assert(sym == "timestamp" && len(tsBin) == 4 && binary.BigEndian.Uint32(tsBin) == 1337, "'timestamp' consistency check failed")
	registerConstraint("timestamp", prefix)
}

func TimestampFromConstraint(data []byte) (uint32, bool) {
	sym, _, args, err := easyfl.ParseBinaryOneLevel(data, 1)
	if err != nil {
		return 0, false
	}
	tsBin := easyfl.StripDataPrefix(args[0])
	if sym != "timestamp" || len(tsBin) != 4 {
		return 0, false
	}
	return binary.BigEndian.Uint32(tsBin), true
}
