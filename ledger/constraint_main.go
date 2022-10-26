package ledger

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

const TimeStampConstraintSource = `
// $0 - 4 bytes Unix seconds big-endian 
func timestamp: or(
	and( isProducedBranch(@), equal($0, txTimestampBytes) ),
	and( isConsumedBranch(@), lessThan($0, txTimestampBytes) )	
)
`

func AmountConstraint(amount uint64) []byte {
	src := fmt.Sprintf("amount(u64/%d)", amount)
	_, _, binCode, err := easyfl.CompileExpression(src)
	easyfl.AssertNoError(err)
	return binCode
}

var (
	amountConstraintPrefix []byte
	amountConstraintLen    int
)

func initAmountConstraint() {
	prefix, err := easyfl.FunctionCodeBytesByName("amount")
	easyfl.AssertNoError(err)
	common.Assert(0 < len(prefix) && len(prefix) <= 2, "0<len(prefix) && len(prefix)<=2")
	template := AmountConstraint(0)
	amountConstraintLen = len(template)
	lenConstraintPrefix := len(prefix) + 1
	common.Assert(len(template) == len(prefix)+1+8, "len(template)==len(prefix)+1+8")
	amountConstraintPrefix = easyfl.Concat(template[:lenConstraintPrefix])
}

func AmountFromConstraint(data []byte) (uint64, error) {
	if len(data) != amountConstraintLen {
		return 0, fmt.Errorf("AmountFromConstraint:: wrong data length")
	}
	if !bytes.HasPrefix(data, amountConstraintPrefix) {
		return 0, fmt.Errorf("AmountFromConstraint:: not an amount constraint")
	}
	return binary.BigEndian.Uint64(data[len(amountConstraintPrefix):]), nil
}

func TimestampConstraint(unixSec uint32) []byte {
	src := fmt.Sprintf("timestamp(u32/%d)", unixSec)
	_, _, binCode, err := easyfl.CompileExpression(src)
	easyfl.AssertNoError(err)
	return binCode
}

var (
	timestampConstraintPrefix []byte
	timestampConstraintLen    int
)

func initTimestampConstraint() {
	prefix, err := easyfl.FunctionCodeBytesByName("timestamp")
	easyfl.AssertNoError(err)
	common.Assert(0 < len(prefix) && len(prefix) <= 2, "0<len(prefix) && len(prefix)<=2")
	template := TimestampConstraint(0)
	timestampConstraintLen = len(template)
	lenConstraintPrefix := len(prefix) + 1
	common.Assert(len(template) == len(prefix)+1+4, "len(template)==len(prefix)+1+4")
	timestampConstraintPrefix = easyfl.Concat(template[:lenConstraintPrefix])
}

func TimestampFromConstraint(data []byte) (uint32, error) {
	if len(data) != timestampConstraintLen {
		return 0, fmt.Errorf("TimestampFromConstraint:: wrong data length")
	}
	if !bytes.HasPrefix(data, timestampConstraintPrefix) {
		return 0, fmt.Errorf("TimestampFromConstraint:: not a timestamp constraint")
	}
	return binary.BigEndian.Uint32(data[len(timestampConstraintPrefix):]), nil
}
