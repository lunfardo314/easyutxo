package constraint

import (
	"encoding/binary"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
)

const amountSource = `
func storageDepositEnough: greaterOrEqualThan(
	$0,
	concat(u32/0, mul16_32(#vbCost16,len16(selfOutputBytes)))
)

// $0 - amount uint64 big-endian
func amount: or(
	isConsumedBranch(@),               // not checked in consumed branch
	and(
		isProducedBranch(@),           // checked in produced branch
		equal(len8($0),8),             // length must be 8
		storageDepositEnough($0)       // must satisfy minimum storage deposit requirements
	)
)
`

func AmountConstraintSource(amount uint64) string {
	return fmt.Sprintf("amount(u64/%d)", amount)
}

func AmountConstraintBin(amount uint64) []byte {
	return mustBinFromSource(AmountConstraintSource(amount))
}

func initAmountConstraint() {
	easyfl.MustExtendMany(amountSource)

	// sanity check
	example := AmountConstraintBin(1337)
	sym, prefix, args, err := easyfl.ParseBinaryOneLevel(example, 1)
	easyfl.AssertNoError(err)
	amountBin := easyfl.StripDataPrefix(args[0])
	common.Assert(sym == "amount" && len(amountBin) == 8 && binary.BigEndian.Uint64(amountBin) == 1337, "'amount' consistency check failed")
	registerConstraint("amount", prefix)
}

func AmountFromConstraint(data []byte) (uint64, bool) {
	sym, _, args, err := easyfl.ParseBinaryOneLevel(data)
	if err != nil {
		return 0, false
	}
	if sym != "amount" {
		return 0, false
	}
	amountBin := easyfl.StripDataPrefix(args[0])
	if len(amountBin) != 8 {
		return 0, false
	}
	return binary.BigEndian.Uint64(easyfl.StripDataPrefix(amountBin)), true
}
