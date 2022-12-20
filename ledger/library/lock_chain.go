package library

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/unitrie/common"
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
	sym, _, args, err := easyfl.ParseBytecodeOneLevel(data, 1)
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

func (cl ChainLock) source() string {
	return fmt.Sprintf(chainLockTemplate, hex.EncodeToString(cl))
}

func (cl ChainLock) Bytes() []byte {
	return mustBinFromSource(cl.source())
}

func (cl ChainLock) IndexableTags() []Accountable {
	return []Accountable{cl}
}

func (cl ChainLock) UnlockableWith(acc AccountID, _ uint32) bool {
	return bytes.Equal(cl.AccountID(), acc)
}

func (cl ChainLock) AccountID() AccountID {
	return cl.Bytes()
}

func (cl ChainLock) Name() string {
	return ChainLockName
}

func (cl ChainLock) String() string {
	return cl.source()
}

func (cl ChainLock) AsLock() Lock {
	return cl
}

func (cl ChainLock) ChainID() []byte {
	return common.Concat([]byte(cl))
}

func NewChainLockUnlockParams(chainOutputIndex, chainConstraintIndex byte) []byte {
	return []byte{chainOutputIndex, chainConstraintIndex}
}

func initChainLockConstraint() {
	easyfl.MustExtendMany(ChainLockConstraintSource)

	example := ChainLockNull()
	chainLockBack, err := ChainLockFromBytes(example.Bytes())
	easyfl.AssertNoError(err)
	easyfl.Assert(Equal(chainLockBack, ChainLockNull()), "inconsistency "+ChainLockName)

	prefix, err := easyfl.ParseBytecodePrefix(example.Bytes())
	easyfl.AssertNoError(err)

	registerConstraint(ChainLockName, prefix, func(data []byte) (Constraint, error) {
		return ChainLockFromBytes(data)
	})
}

const ChainLockConstraintSource = `

func selfReferencedChainData :
	parseBytecodeArg(
		consumedConstraintByIndex(selfUnlockParameters),
		#chain,
		0
	)

// $0 - parsed referenced chain constraint
func selfReferencedChainIDAdjusted : if(
	isZero($0),
	blake2b(inputIDByIndex(byte(selfUnlockParameters, 0))),
	$0
)

// $0 - chainID
func validChainUnlock : and(
	equal($0, selfReferencedChainIDAdjusted(slice(selfReferencedChainData,0,31))), // chain id must be equal to the referenced chain id 
	equal(
		// the chain must be unlocked for state transition (mode = 0) 
		byte(unlockParamsByConstraintIndex(selfUnlockParameters),2),
		0
	)
)

func chainLock : and(
	equal(selfBlockIndex,2), // locks must be at block 2
	or(
		and(
			selfIsProducedOutput, 
			equal(len8($0),32)
		),
		and(
			selfIsConsumedOutput,
			not(equal(selfOutputIndex, byte(selfUnlockParameters,0))), // prevent self referencing 
			validChainUnlock($0)
		)
	)
)

`
