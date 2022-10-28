package constraint

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
)

const AmountConstraintSource = `
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

func Amount(amount uint64) []byte {
	src := fmt.Sprintf("amount(u64/%d)", amount)
	_, _, binCode, err := easyfl.CompileExpression(src)
	easyfl.AssertNoError(err)
	return binCode
}

func initAmountConstraint() {
	easyfl.MustExtendMany(AmountConstraintSource)

	example := Amount(1337)
	prefix, args, err := easyfl.ParseCallWithConstants(example, 1)
	easyfl.AssertNoError(err)
	common.Assert(len(args[0]) == 8 && binary.BigEndian.Uint64(args[0]) == 1337, "len(args)==8 && binary.BigEndian.Uint64(args[0])==1337")
	registerConstraint("amount", prefix)
}

func AmountFromConstraint(data []byte) (uint64, error) {
	prefix, args, err := easyfl.ParseCallWithConstants(data, 1)
	if err != nil {
		return 0, err
	}
	prefix1, ok := PrefixByName("amount")
	common.Assert(ok, "no amount")
	if !bytes.Equal(prefix, prefix1) {
		return 0, fmt.Errorf("AmountFromConstraint:: not an 'amount' constraint")
	}
	if len(args[0]) != 8 {
		return 0, fmt.Errorf("AmountFromConstraint:: wrong data length")
	}
	return binary.BigEndian.Uint64(args[0]), nil
}
