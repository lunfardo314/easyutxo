package ledger

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger/constraints"
	"github.com/lunfardo314/unitrie/common"
	"github.com/lunfardo314/unitrie/models/trie_blake2b"
)

// commitment model singleton

var CommitmentModel = trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256)

type GenesisDataStruct struct {
	StateIdentity       []byte
	Address             constraints.AddressED25519
	Timestamp           uint32
	InitialSupply       uint64
	OutputID            OutputID
	MilestoneController constraints.AddressED25519
	TimestampMilestone  uint32
	MilestoneOutputID   OutputID
	MilestoneDeposit    uint64
	MilestoneChainID    [32]byte
}

const milestoneDeposit = 1000

// genesis related types and constants

var (
	genesisOutputID, genesisMilestoneOutputID OutputID
	MilestoneChainID                          [32]byte
)

func init() {
	genesisOutputID = OutputID{}
	s, err := hex.DecodeString(strings.Repeat("ff", OutputIDLength))
	if err != nil {
		panic(err)
	}
	copy(genesisMilestoneOutputID[:], s)
	copy(MilestoneChainID[:], make([]byte, 32)) // all 0
}

func GenesisData(identity []byte, genesisAddr, milestoneController constraints.AddressED25519, initialSupply uint64, ts uint32) *GenesisDataStruct {
	easyfl.Assert(initialSupply > 0, "initialSupply > 0")
	ret := &GenesisDataStruct{
		StateIdentity:       identity,
		Address:             genesisAddr,
		Timestamp:           ts,
		InitialSupply:       initialSupply,
		OutputID:            genesisOutputID,
		MilestoneController: milestoneController,
		TimestampMilestone:  ts + 1,
		MilestoneOutputID:   genesisMilestoneOutputID,
		MilestoneDeposit:    milestoneDeposit,
		MilestoneChainID:    MilestoneChainID,
	}

	return ret
}

func (g *GenesisDataStruct) String() string {
	return "Genesis data:\n" +
		fmt.Sprintf("   Address: %s\n", g.Address.String()) +
		fmt.Sprintf("   Timestamp: %d\n", g.Timestamp) +
		fmt.Sprintf("   Initial supply: %d\n", g.InitialSupply) +
		fmt.Sprintf("   OutputID: %s\n", g.OutputID.String()) +
		fmt.Sprintf("   Milestone controller: %s\n", g.MilestoneController.String()) +
		fmt.Sprintf("   Timestamp (milestone): %d\n", g.TimestampMilestone) +
		fmt.Sprintf("   Milestone outputID: %s\n", g.MilestoneOutputID.String()) +
		fmt.Sprintf("   Milestone deposit: %d\n", g.MilestoneDeposit) +
		fmt.Sprintf("   Milestone chainID: %s\n", easyfl.Fmt(g.MilestoneChainID[:]))
}
