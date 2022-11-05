package library

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/lunfardo314/easyfl"
)

/*
 ChainConstraint constraint imposes chain of consumed UTXOs with the same identity from the origin to the final state
 Each chain represents a sequence of state changes.
 Structure of the output:
 - identity (chainID)
 - amount
 - timestamp
 - chain lock: 2 locks: state lock and governance lock
 - state metadata
 - governance metadata
 - immutable metadata

- ChainConstraint data constraint: array: chain config (back ref), chain identity, state metadata, governance metadata, immutable metadata (xN)
- ChainConstraint lock constraint: state controller, governance controller
- ChainConstraint data unlock params: forward ref
*/

const (
	ChainModeTransitionModeState = byte(iota)
	ChainTransitionModeGovernance
	ChainTransitionModeOrigin
)

// ChainConstraint is chain constraint
type ChainConstraint struct {
	// ID all-0 for origin
	ID             [32]byte
	TransitionMode byte
	// Previous index of the consumed chain input with the same ID. Must be 0xFF for the origin
	PreviousOutput byte
	PreviousBlock  byte
}

const (
	ChainConstraintName     = "chain"
	chainConstraintTemplate = ChainConstraintName + "(0x%s)"
)

func NewChainConstraint(id [32]byte, mode, prevOut, prevBlock byte) *ChainConstraint {
	return &ChainConstraint{
		ID:             id,
		TransitionMode: mode,
		PreviousOutput: prevOut,
		PreviousBlock:  prevBlock,
	}
}

func NewChainOrigin() *ChainConstraint {
	return &ChainConstraint{
		PreviousOutput: 0xff,
		PreviousBlock:  0xff,
		TransitionMode: ChainTransitionModeOrigin,
	}
}

func (ch *ChainConstraint) Name() string {
	return ChainConstraintName
}

func (ch *ChainConstraint) Bytes() []byte {
	return mustBinFromSource(ch.source())
}

func (ch *ChainConstraint) String() string {
	return ch.source()
}

func (ch *ChainConstraint) source() string {
	return fmt.Sprintf(chainConstraintTemplate,
		hex.EncodeToString(easyfl.Concat(ch.ID[:], ch.PreviousOutput, ch.PreviousBlock, ch.TransitionMode)))
}

func ChainConstraintFromBytes(data []byte) (*ChainConstraint, error) {
	sym, _, args, err := easyfl.ParseBinaryOneLevel(data, 1)
	if err != nil {
		return nil, err
	}
	if sym != ChainConstraintName {
		return nil, fmt.Errorf("not a chain constraint")
	}
	constraintData := easyfl.StripDataPrefix(args[0])
	if len(constraintData) != 32+3 {
		return nil, fmt.Errorf("wrong data len")
	}
	ret := &ChainConstraint{
		PreviousOutput: constraintData[32],
		PreviousBlock:  constraintData[33],
		TransitionMode: constraintData[34],
	}
	copy(ret.ID[:], constraintData[:32])

	return ret, nil
}

func initChainConstraint() {
	easyfl.MustExtendMany(chainConstraintSource)

	example := NewChainOrigin()
	back, err := ChainConstraintFromBytes(example.Bytes())
	easyfl.AssertNoError(err)
	easyfl.Assert(bytes.Equal(back.Bytes(), example.Bytes()), "inconsistency in "+ChainConstraintName)

	_, prefix, _, err := easyfl.ParseBinaryOneLevel(example.Bytes(), 1)
	easyfl.AssertNoError(err)

	registerConstraint(ChainConstraintName, prefix, func(data []byte) (Constraint, error) {
		return ChainConstraintFromBytes(data)
	})
}

const chainConstraintSource = `

// parsing constraint data
func originChainData: concat(repeat(0,32), 0xffffff)

func chainID : slice($0, 0, 31)

func transitionMode: byte($0, 34)

func predecessorConstraintIndex : slice($0, 32, 33)

func predecessorOutput : consumedConstraintByIndex(predecessorConstraintIndex($0))

func predecessorInputID : inputIDByIndex(byte($0,32))

// transition mode constants
func chainTransitionModeState : 0
func chainTransitionModeGovernance : 1
func chainTransitionModeOrigin : 2

// unlock parameters and predecessor reference 3 bytes: 
// 0 - output index 
// 1 - block in the referenced output
// 2 - transitionMode 

// for produced output
// $0 - self produced constraint data
// $1 - predecessor data
func validPredecessorData : or(
	and(
		// predecessor is origin, 
		equal($1, originChainData),
		equal(chainID($0), blake2b(predecessorInputID($0)))
	),
	equal(chainID($0), chainID($1))
)

// $0 - predecessor constraint index
func chainPredecessorData: 
	parseCallArg(
		consumedConstraintByIndex($0),
		selfCallPrefix,
		0
	)

// $0 - self chain data (consumed)
// $1 - successor constraint parsed data (produced)
func validUnlockSuccessorData : and(
	equal(chainID($0),chainID($1)),
	equal(predecessorConstraintIndex($1), selfConstraintIndex)
)

func chainSuccessorData : 
	parseCallArg(
		producedConstraintByIndex(slice(selfUnlockParameters,0,1)),
		selfCallPrefix,
		0
	)

// $0 - 35 bytes chain data 
func chainTransition : or(
	and(
		// 'consumed' side case, checking if successor is valid
		isPathToConsumedOutput(@),
		validUnlockSuccessorData($0, chainSuccessorData)
	), 
	and(
		// 'produced' side case
		isPathToProducedOutput(@),
		validPredecessorData($0, chainPredecessorData( predecessorConstraintIndex($0) ))
	)
)

// $0 - 35 bytes: 32 bytes chain id, 1 byte predecessor output ID, 1 byte predecessor block id, 1 byte transition mode 
func chain: and(
	not(equal(selfOutputIndex, 0xff)),  // chain constraint cannot be on output with index 0xff
	or(
		and( // chain origin
			isPathToProducedOutput(@),
			equal($0, originChainData)  // enforced reserved values at origin
		),
		// transition
		chainTransition($0)
	)
)

`
