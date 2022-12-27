package state

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraints"
	"github.com/lunfardo314/unitrie/common"
)

type GenesisDataStruct struct {
	Address             constraints.AddressED25519
	Timestamp           uint32
	InitialSupply       uint64
	OutputID            ledger.OutputID
	MilestoneController constraints.AddressED25519
	TimestampMilestone  uint32
	MilestoneOutputID   ledger.OutputID
	MilestoneDeposit    uint64
	MilestoneChainID    [32]byte
	StateCommitment     common.VCommitment
}

const milestoneDeposit = 1000

// genesis related types and constants

var (
	genesisOutputID, genesisMilestoneOutputID ledger.OutputID
	milestoneChainID                          [32]byte
)

func init() {
	genesisOutputID = ledger.OutputID{}
	s, err := hex.DecodeString(strings.Repeat("ff", ledger.OutputIDLength))
	if err != nil {
		panic(err)
	}
	copy(genesisMilestoneOutputID[:], s)
	copy(milestoneChainID[:], make([]byte, 32)) // all 0
}

func GetGenesisData(genesisAddr, milestoneController constraints.AddressED25519, initialSupply uint64, ts uint32) *GenesisDataStruct {
	easyfl.Assert(initialSupply > 0, "initialSupply > 0")
	ret := &GenesisDataStruct{
		Address:             genesisAddr,
		Timestamp:           ts,
		InitialSupply:       initialSupply,
		OutputID:            genesisOutputID,
		MilestoneController: milestoneController,
		TimestampMilestone:  ts + 1,
		MilestoneOutputID:   genesisMilestoneOutputID,
		MilestoneDeposit:    milestoneDeposit,
		MilestoneChainID:    milestoneChainID,
		StateCommitment:     nil,
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
		fmt.Sprintf("   Milestone chainID: %s\n", easyfl.Fmt(g.MilestoneChainID[:])) +
		fmt.Sprintf("   State commitment: %s\n", g.StateCommitment)
}

func (g *GenesisDataStruct) genesisOutput() []byte {
	amount := constraints.NewAmount(g.InitialSupply) //  - g.MilestoneDeposit)
	timestamp := constraints.NewTimestamp(g.Timestamp)
	ret := lazyslice.EmptyArray()
	ret.Push(amount.Bytes())
	ret.Push(timestamp.Bytes())
	ret.Push(g.Address.Bytes())
	return ret.Bytes()
}

func (g *GenesisDataStruct) genesisMilestoneOutput() []byte {
	amount := constraints.NewAmount(g.MilestoneDeposit)
	timestamp := constraints.NewTimestamp(g.TimestampMilestone)
	chainConstraint := constraints.NewChainConstraint(g.MilestoneChainID, 0, 0, 0)
	stateCommitment, err := constraints.NewGeneralScriptFromSource(fmt.Sprintf("id(0x%s)", g.StateCommitment.String()))
	common.AssertNoError(err)

	ret := lazyslice.EmptyArray()
	ret.Push(amount.Bytes())
	ret.Push(timestamp.Bytes())
	ret.Push(g.MilestoneController.Bytes())
	ret.Push(chainConstraint.Bytes())
	ret.Push(stateCommitment.Bytes())
	return ret.Bytes()

}
