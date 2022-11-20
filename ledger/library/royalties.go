package library

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/lunfardo314/easyfl"
)

// the ImmutableData constraint forces sending specified amount of tokens to specified address

type RoyaltiesED25519 struct {
	Address AddressED25519
	Amount  uint64
}

const (
	RoyaltiesED25519Name     = "royaltiesED25519"
	royaltiesED25519Template = RoyaltiesED25519Name + "(0x%s, u64/%d)"
)

func NewRoyalties(addr AddressED25519, amount uint64) *RoyaltiesED25519 {
	return &RoyaltiesED25519{
		Address: addr,
		Amount:  amount,
	}
}

func RoyaltiesED25519FromBytes(data []byte) (*RoyaltiesED25519, error) {
	sym, _, args, err := easyfl.ParseBytecodeOneLevel(data, 2)
	if err != nil {
		return nil, err
	}
	if sym != RoyaltiesED25519Name {
		return nil, fmt.Errorf("not a ImmutableData")
	}
	addrBin := easyfl.StripDataPrefix(args[0])
	addr, err := AddressED25519FromBytes(addrBin)
	if err != nil {
		return nil, err
	}
	amountBin := easyfl.StripDataPrefix(args[1])
	if len(amountBin) != 8 {
		return nil, fmt.Errorf("wrong amount")
	}
	return NewRoyalties(addr, binary.BigEndian.Uint64(amountBin)), nil
}

func (cl *RoyaltiesED25519) source() string {
	return fmt.Sprintf(royaltiesED25519Template, hex.EncodeToString(cl.Address.Bytes()), cl.Amount)
}

func (cl *RoyaltiesED25519) Bytes() []byte {
	return mustBinFromSource(cl.source())
}

func (cl *RoyaltiesED25519) Name() string {
	return RoyaltiesED25519Name
}

func (cl RoyaltiesED25519) String() string {
	return cl.source()
}

func initChainRoyaltiesConstraint() {
	easyfl.MustExtendMany(RoyaltiesED25519Source)

	addr0 := AddressED25519Null()
	example := NewRoyalties(addr0, 1337)
	royaltiesBack, err := RoyaltiesED25519FromBytes(example.Bytes())
	easyfl.AssertNoError(err)
	easyfl.Assert(Equal(royaltiesBack.Address, addr0), "inconsistency "+RoyaltiesED25519Name)
	easyfl.Assert(royaltiesBack.Amount == 1337, "inconsistency "+RoyaltiesED25519Name)

	prefix, err := easyfl.ParseCallPrefixFromBytecode(example.Bytes())
	easyfl.AssertNoError(err)

	registerConstraint(RoyaltiesED25519Name, prefix, func(data []byte) (Constraint, error) {
		return RoyaltiesED25519FromBytes(data)
	})
}

// Unlock params must point to the output which sends at least specified amount of tokens to the same address
// where the sender is locked

const RoyaltiesED25519Source = `
func royaltiesED25519 : or(
	selfIsProducedOutput,  // always satisfied if produced
	and(
		selfIsConsumedOutput,
		equal(
			$0,
			lockConstraint(producedOutputByIndex(selfUnlockParameters))
		),
		lessOrEqualThan(
			$1,
			amountValue(producedOutputByIndex(selfUnlockParameters))
		)
	),
	!!!royaltiesED25519_constraint_failed
)
`