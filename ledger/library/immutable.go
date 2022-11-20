package library

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/lunfardo314/easyfl"
)

// the ImmutableData constraint forces teh same data to be repeated on the successor of the chain

type ImmutableData struct {
	Data            []byte
	ChainBlockIndex byte
}

const (
	ImmutableDataName     = "immutableData"
	immutableDataTemplate = ImmutableDataName + "(0x%s, %d)"
)

func NewImmutableData(data []byte, chainBlockIndex byte) *ImmutableData {
	return &ImmutableData{
		Data:            data,
		ChainBlockIndex: chainBlockIndex,
	}
}

func ImmutableDataFromBytes(data []byte) (*ImmutableData, error) {
	sym, _, args, err := easyfl.ParseBytecodeOneLevel(data, 2)
	if err != nil {
		return nil, err
	}
	if sym != ImmutableDataName {
		return nil, fmt.Errorf("not a ImmutableData")
	}
	d := easyfl.StripDataPrefix(args[0])
	if err != nil {
		return nil, err
	}
	idxBin := easyfl.StripDataPrefix(args[1])
	if len(idxBin) != 1 {
		return nil, fmt.Errorf("wrong chain block index")
	}
	return NewImmutableData(d, idxBin[0]), nil
}

func (d *ImmutableData) source() string {
	return fmt.Sprintf(immutableDataTemplate, hex.EncodeToString(d.Data), d.ChainBlockIndex)
}

func (d *ImmutableData) Bytes() []byte {
	return mustBinFromSource(d.source())
}

func (d *ImmutableData) Name() string {
	return ImmutableDataName
}

func (d *ImmutableData) String() string {
	return d.source()
}

func initImmutableDataConstraint() {
	easyfl.MustExtendMany(ImmutableDataSource)

	data := []byte("some data")
	example := NewImmutableData(data, 5)
	immutableDataBack, err := ImmutableDataFromBytes(example.Bytes())
	easyfl.AssertNoError(err)
	easyfl.Assert(bytes.Equal(data, immutableDataBack.Data), "inconsistency "+ImmutableDataName)
	easyfl.Assert(immutableDataBack.ChainBlockIndex == 5, "inconsistency "+ImmutableDataName)

	prefix, err := easyfl.ParseCallPrefixFromBytecode(example.Bytes())
	easyfl.AssertNoError(err)

	registerConstraint(ImmutableDataName, prefix, func(data []byte) (Constraint, error) {
		return ImmutableDataFromBytes(data)
	})
}

// Unlock params must point to the output which sends at least specified amount of tokens to the same address
// where the sender is locked

const ImmutableDataSource = `

// $0 - the data
// $1 - index of the chain block
func immutableData : or(
	and(
		selfIsProducedOutput,
		equal(parseCallPrefix(selfSiblingBlock($1)), #chain), // $1 must point to the sibling-chain constraint
	),
	and(
		selfIsConsumedOutput,
		$0 // the successor must have the same data TODO
	),
	!!!immutableData_constraint_failed
)
`
