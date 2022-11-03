package library

/*
 Chain constraint imposes chain of consumed UTXOs with the same identity from the origin to the final state
 Each chain represents a sequence of state changes.
 Structure of the output:
 - identity (chainID)
 - amount
 - timestamp
 - chain lock: 2 locks: state lock and governance lock
 - state metadata
 - governance metadata
 - immutable metadata

- Chain data constraint: array: chain config (back ref), chain identity, state metadata, governance metadata, immutable metadata (xN)
- Chain lock constraint: state controller, governance controller
- Chain data unlock params: forward ref
*/

type ChainHeader struct {
	// ID all-0 for origin
	ID [32]byte
	// Previous index of the consumed chain input with the same ID. Must be 0xFF for the origin
	Previous byte
	// Governance is true for governance transition, false for state transition
	Governance bool
	// Incremental state index
	StateIndex uint32
}

type ChainStateMetadata struct {
	ChainHeaderBlockIndex byte
	PreviousBlockIndex    byte
	Data                  []byte
}

type ChainGovernanceMetadata struct {
	ChainHeaderBlockIndex byte
	PreviousBlockIndex    byte
	Data                  []byte
}

type ChainImmutableMetadata struct {
	ChainHeaderBlockIndex byte
	PreviousBlockIndex    byte
	Data                  []byte
}
