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

const (
	amountName     = "amount"
	amountTemplate = amountName + "(u64/%d)"
)

type Amount uint64

func (a Amount) Name() string {
	return amountName
}

func (a Amount) source() string {
	return fmt.Sprintf(amountTemplate, uint64(a))
}

func (a Amount) Bytes() []byte {
	return mustBinFromSource(a.source())
}

func (a Amount) String() string {
	return fmt.Sprintf("%s(%d)", amountName, uint64(a))
}

func NewAmount(a uint64) Amount {
	return Amount(a)
}

func initAmountConstraint() {
	easyfl.MustExtendMany(amountSource)
	// sanity check
	example := NewAmount(1337)
	sym, prefix, args, err := easyfl.ParseBinaryOneLevel(example.Bytes(), 1)
	easyfl.AssertNoError(err)
	amountBin := easyfl.StripDataPrefix(args[0])
	common.Assert(sym == amountName && len(amountBin) == 8 && binary.BigEndian.Uint64(amountBin) == 1337, "'amount' consistency check failed")
	registerConstraint(amountName, prefix, func(data []byte) (Constraint, error) {
		return AmountFromBytes(data)
	})
}

func AmountFromBytes(data []byte) (Amount, error) {
	sym, _, args, err := easyfl.ParseBinaryOneLevel(data)
	if err != nil {
		return 0, err
	}
	if sym != amountName {
		return 0, fmt.Errorf("not an 'amount' constraint")
	}
	amountBin := easyfl.StripDataPrefix(args[0])
	if len(amountBin) != 8 {
		return 0, fmt.Errorf("wrong data length")
	}
	return Amount(binary.BigEndian.Uint64(amountBin)), nil
}
