package library

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/iotaledger/trie.go/common"
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
	chainConstraintTemplate = ChainConstraintName + "(0x%s,0x%s)"
)

func NewChainConstraint(id [32]byte, mode, prevOut, prevBlock byte) *ChainConstraint {
	return &ChainConstraint{
		ID:             id,
		TransitionMode: mode,
		PreviousOutput: prevOut,
		PreviousBlock:  prevBlock,
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
		hex.EncodeToString(ch.ID[:]), hex.EncodeToString(easyfl.Concat(ch.TransitionMode, ch.PreviousOutput, ch.PreviousBlock)))
}

func ChainConstraintFromBytes(data []byte) (*ChainConstraint, error) {
	sym, _, args, err := easyfl.ParseBinaryOneLevel(data, 2)
	if err != nil {
		return nil, err
	}
	if sym != ChainConstraintName {
		return nil, fmt.Errorf("not a chain constraint")
	}
	id := easyfl.StripDataPrefix(args[0])
	ref := easyfl.StripDataPrefix(args[1])
	if len(id) != 32 || len(ref) != 3 {
		return nil, fmt.Errorf("wrong data len")
	}
	ret := &ChainConstraint{
		TransitionMode: ref[0],
		PreviousOutput: ref[1],
		PreviousBlock:  ref[2],
	}
	copy(ret.ID[:], id)

	return ret, nil
}

func initChainConstraint() {
	easyfl.MustExtendMany(chainConstraintSource)

	example := NewChainConstraint([32]byte{}, 0, 0, 0)
	sym, prefix, args, err := easyfl.ParseBinaryOneLevel(example.Bytes(), 2)
	easyfl.AssertNoError(err)
	id := easyfl.StripDataPrefix(args[0])
	ref := easyfl.StripDataPrefix(args[1])
	common.Assert(sym == ChainConstraintName, "inconsistency in 'chain constraint' 1")
	common.Assert(bytes.Equal(id, make([]byte, 32)), "inconsistency in 'chain constraint' 2")
	common.Assert(bytes.Equal(ref, []byte{0, 0, 0}), "inconsistency in 'chain constraint' 3")

	registerConstraint(ChainConstraintName, prefix, func(data []byte) (Constraint, error) {
		return ChainConstraintFromBytes(data)
	})
}

const chainConstraintSource = `

// transition mode constants
func chainTransitionModeState : 0
func chainTransitionModeGovernance : 1
func chainTransitionModeOrigin : 2

// unlock parameters and predecessor reference 3 bytes: 
// 0 - transitionMode 
// 1 - output index 
// 2 - block in the referenced output

// $0 - chain id
// $1 - predecessor ref 2 bytes (output, block)
func validPredecessor : equal(
	$0,
	parseCallArg(
		consumedBlockByOutputIndex(
			byte($1, 1), 
			byte($1, 2)
		),
		selfCallPrefix,
		0
	)
)

// $0 - chain id 
// must correctly point to the same constraint and the same chain id
// only makes sense for the 'consumed' side
func validSuccessor : equal(
	$0,
	parseCallArg(
		producedBlockByOutputIndex(
			byte(selfUnlockParameters, 1), 
			byte(selfUnlockParameters, 2)
		),
		selfCallPrefix,
		0
	)
)

// $0 - id
// $1 - mode/predecessor ref 3 bytes
func chainTransition : or(
	and(
		// 'consumed' side case, checking if successor is valid
		isPathToConsumedOutput(@),
		validSuccessor($0)
	), 
	and(
		// 'produced' side case
		isPathToProducedOutput(@),
		validPredecessor($0, $1)
	)
)

// $0 - 32 byte chain id
// $1 - 3- byte transition mode/predecessor reference concat(transitionMode, outputIdx, blockIdx)
func chain: or(
	and( // chain origin
		isPathToProducedOutput(@),
		equal(byte($1,0), chainTransitionModeOrigin),
		equal($0, repeat(0,32)),         // chain id must be 32 byte 0s
		equal($1, 0xffff)                // predecessor ref must be reserved value of 0xffff
	),
	// transition
	chainTransition($0, $1)
)	

`
