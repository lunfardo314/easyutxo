package constraints

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/unitrie/common"
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

func NewChainConstraint(id [32]byte, prevOut, prevBlock, mode byte) *ChainConstraint {
	return &ChainConstraint{
		ID:             id,
		TransitionMode: mode,
		PreviousOutput: prevOut,
		PreviousBlock:  prevBlock,
	}
}

func NewChainOrigin() *ChainConstraint {
	return NewChainConstraint([32]byte{}, 0xff, 0xff, 0xff)
}

func (ch *ChainConstraint) IsOrigin() bool {
	if ch.ID != [32]byte{} {
		return false
	}
	if ch.PreviousOutput != 0xff {
		return false
	}
	if ch.PreviousBlock != 0xff {
		return false
	}
	if ch.TransitionMode != 0xff {
		return false
	}
	return true
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
		hex.EncodeToString(common.Concat(ch.ID[:], ch.PreviousOutput, ch.PreviousBlock, ch.TransitionMode)))
}

func (ch *ChainConstraint) ChainLock() ChainLock {
	ret, err := ChainLockFromChainID(ch.ID[:])
	easyfl.AssertNoError(err)
	return ret
}

func ChainConstraintFromBytes(data []byte) (*ChainConstraint, error) {
	sym, _, args, err := easyfl.ParseBytecodeOneLevel(data, 1)
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

	_, prefix, _, err := easyfl.ParseBytecodeOneLevel(example.Bytes(), 1)
	easyfl.AssertNoError(err)

	registerConstraint(ChainConstraintName, prefix, func(data []byte) (Constraint, error) {
		return ChainConstraintFromBytes(data)
	})
}

const chainConstraintSource = `
// chain(<chain constraint data>)
// <chain constraint data: 35 bytes:
// - 0-31 bytes chain id 
// - 32 byte predecessor output index 
// - 33 byte predecessor block index 
// - 34 byte transition mode 

// reserved value of the chain constraint data at origin
func originChainData: concat(repeat(0,32), 0xffffff)
func destroyUnlockParams : 0xffffff

// parsing chain constraint data
// $0 - chain constraint data
func chainID : slice($0, 0, 31)
func transitionMode: byte($0, 34)
func predecessorConstraintIndex : slice($0, 32, 33) // 2 bytes

// accessing to predecessor data
func predecessorOutput : consumedConstraintByIndex(predecessorConstraintIndex($0))
func predecessorInputID : inputIDByIndex(byte($0,32))

// unlock parameters for the chain constraint. 3 bytes: 
// 0 - successor output index 
// 1 - successor block index
// 2 - transition mode must be equal to the transition mode in the successor constrain data 

// only called for produced output
// $0 - self produced constraint data
// $1 - predecessor data
func validPredecessorData : and(
	if(
		isZero(chainID($1)), 
		and(
			// case 1: predecessor is origin. ChainID must be blake2b hash of the corresponding input ID 
			equal($1, originChainData),
			equal(chainID($0), blake2b(predecessorInputID($0)))
		),
		and(
			// case 2: normal transition
			equal(chainID($0), chainID($1)),
		)
	),
	equal(
		// enforcing equal transition mode on unlock data and on the produced output
		transitionMode($0),
		byte(unlockParamsByConstraintIndex(predecessorConstraintIndex($0)),2)
	)
)

// $0 - predecessor constraint index
func chainPredecessorData: 
	parseBytecodeArg(
		consumedConstraintByIndex($0),
		selfBytecodePrefix,
		0
	)

// $0 - self chain data (consumed)
// $1 - successor constraint parsed data (produced)
func validSuccessorData : and(
		if (
			// if chainID = 0, it must be origin data
			// otherwise chain IDs must be equal on both sides
			isZero(chainID($0)),
			equal($0, originChainData),
			equal(chainID($0),chainID($1))
		),
		// the successor (produced) must point to the consumed (self)
		equal(predecessorConstraintIndex($1), selfConstraintIndex)
)

// chain successor data is computed form in the context of the consumed output
// from the selfUnlock data
func chainSuccessorData : 
	parseBytecodeArg(
		producedConstraintByIndex(slice(selfUnlockParameters,0,1)),
		selfBytecodePrefix,
		0
	)

// Constraint source: chain($0)
// $0 - 35-bytes data: 
//     32 bytes chain id
//     1 byte predecessor output index 
//     1 byte predecessor block index
//     1 byte transition mode
// Transition mode: 
//     0x00 - state transition
//     0xff - origin state, can be any other values. 
// It is enforced by the chain constraint 
// but it is interpreted by other constraints, bound to chain 
// constraint, such as controller locks
func chain: and(
      // chain constraint cannot be on output with index 0xff = 255
   not(equal(selfOutputIndex, 0xff)),  
   or(
      if(
        // if it is produced output with zero-chainID, it is chain origin.
         and(
            isZero(chainID($0)),
            selfIsProducedOutput
         ),
         or(
            // enforcing valid constraint data of the origin: concat(repeat(0,32), 0xffffff)
            equal($0, originChainData), 
            !!!chain_wrong_origin
         ),
         nil
       ),
        // check validity of chain transition. Unlock data of the constraint 
        // must point to the valid successor (in case of consumed output) 
        // or predecessor (in case of produced output) 
       and(
           // 'consumed' side case, checking if unlock params and successor is valid
          selfIsConsumedOutput,
          or(
               // consumed chain output is being destroyed (no successor)
            equal(selfUnlockParameters, destroyUnlockParams),
               // or it must be unlocked by pointing to the successor
            validSuccessorData($0, chainSuccessorData),     
            !!!chain_wrong_successor
          )	
       ), 
       and(
          // 'produced' side case, checking if predecessor is valid
           selfIsProducedOutput,
           or(
              // 'produced' side case checking if predecessor is valid
              validPredecessorData($0, chainPredecessorData( predecessorConstraintIndex($0) )),
              !!!chain_wrong_predecessor
           )
       ),
       !!!chain_constraint_failed
   )
)
`
