package easyutxo

import "io"

type Transaction struct {
	Essence     TransactionEssence
	UnlockBlock UnlockBlocks
}

type TransactionEssence struct {
	Inputs       Inputs
	Outputs      Outputs
	DataElements DataElements
}

type UnlockBlocks []UnlockBlock
type UnlockBlock []byte

type DataElements []DataElement
type DataElement []byte

type Inputs []Input
type Input []byte

type Outputs []Output

type Output struct {
	Assets OutputAssets
	Blocks OutputBlocks
}

type OutputAssets []byte

type OutputBlocks []OutputBlock

type OutputBlock struct {
	Invocation Invocation
	//Script     Script
}

// Invocation
//   0 inline, the script and bytes is right after
//   1 script attached, next 32 or 20 bytes are hash of the script | DEAS of the script
//   other - well known embeded scripts for

type Invocation interface {
	Read(r io.Reader) error
	Write(w io.Writer) error
}
