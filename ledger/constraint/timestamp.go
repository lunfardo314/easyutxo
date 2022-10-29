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
	sym, prefix, args, err := easyfl.DecompileBinaryOneLevel(example, 1)
	easyfl.AssertNoError(err)
	common.Assert(sym == "timestamp" && len(args[0]) == 4 && binary.BigEndian.Uint32(args[0]) == 1337, "'timestamp' consistency check failed")
	registerConstraint("timestamp", prefix)
}

func TimestampFromConstraint(data []byte) (uint32, bool) {
	sym, _, args, err := easyfl.DecompileBinaryOneLevel(data, 1)
	if err != nil {
		return 0, false
	}
	if sym != "timestamp" || len(args[0]) != 4 {
		return 0, false
	}
	return binary.BigEndian.Uint32(args[0]), true
}
