package library

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/lunfardo314/easyfl"
)

type ChainLock []byte

const (
	ChainLockName     = "chainLock"
	chainLockTemplate = ChainLockName + "(0x%s)"
)

func ChainLockFromChainID(chainID []byte) (ChainLock, error) {
	if len(chainID) != 32 {
		return nil, fmt.Errorf("wrong chainID")
	}
	return chainID, nil
}

func ChainLockNull() ChainLock {
	return make([]byte, 32)
}

func ChainLockFromBytes(data []byte) (ChainLock, error) {
	sym, _, args, err := easyfl.ParseBinaryOneLevel(data, 1)
	if err != nil {
		return nil, err
	}
	if sym != ChainLockName {
		return nil, fmt.Errorf("not a ChainLock")
	}
	chainIdBin := easyfl.StripDataPrefix(args[0])
	if len(chainIdBin) != 32 {
		return nil, fmt.Errorf("wrong data length")
	}
	return chainIdBin, nil
}

func (a ChainLock) source() string {
	return fmt.Sprintf(chainLockTemplate, hex.EncodeToString(a))
}

func (a ChainLock) Bytes() []byte {
	return mustBinFromSource(a.source())
}

func (a ChainLock) IndexableTags() []Accountable {
	return []Accountable{a}
}

func (a ChainLock) UnlockableWith(acc AccountID, ts uint32) bool {
	return bytes.Equal(a.AccountID(), acc)
}

func (a ChainLock) AccountID() AccountID {
	return a.Bytes()
}

func (a ChainLock) Name() string {
	return ChainLockName
}

func (a ChainLock) String() string {
	return a.source()
}

func NewChainLockUnlockParamData(chainOutputIndex, chainConstraintIndex byte) []byte {
	return []byte{chainOutputIndex, chainConstraintIndex}
}

func initChainLockConstraint() {
	easyfl.MustExtendMany(ChainLockConstraintSource)

	example := ChainLockNull()
	chainLockBack, err := ChainLockFromBytes(example.Bytes())
	easyfl.AssertNoError(err)
	easyfl.Assert(Equal(chainLockBack, ChainLockNull()), "inconsistency "+ChainLockName)

	prefix, err := easyfl.ParseCallPrefixFromBinary(example.Bytes())
	easyfl.AssertNoError(err)

	registerConstraint(ChainLockName, prefix, func(data []byte) (Constraint, error) {
		return ChainLockFromBytes(data)
	})
}

const ChainLockConstraintSource = `

func selfReferencedChainID : 
	slice(
		parseCallArg(
			consumedConstraintByIndex(selfUnlockParameters),
			#chain,
			0
		),
	0,
	31
)

func chainLock : and(
	equal(selfBlockIndex,2), // locks must be at block 2
	or(
		and(
			isPathToProducedOutput(@), 
			equal(len8($0),32)
		),
		and(
			isPathToConsumedOutput(@),
			not(equal(selfOutputIndex, byte(selfUnlockParameters,0))), // prevent self referencing 
			equal($0, selfReferencedChainID)
		)
	)
)

`
