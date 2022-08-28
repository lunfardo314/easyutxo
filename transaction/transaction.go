package transaction

import (
	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

const (
	TransactionIDLength = 32
	OutputIDLength      = TransactionIDLength + 2
)

type Transaction struct {
	ls *lazyslice.Tree
}

func NewTransaction() *Transaction {
	ret := &Transaction{ls: lazyslice.TreeEmpty()}
	ret.ls.PushNewSubtreeAtPath() // input groups
	ret.ls.PushNewSubtreeAtPath() // param groups
	ret.ls.PushNewSubtreeAtPath() // output contexts
	ret.ls.PushNewSubtreeAtPath() // transaction level data elements
	return ret
}

func (tx *Transaction) Bytes() []byte {
	return tx.ls.Bytes()
}

type OutputBlock struct {
	Script Script
	Params lazyslice.Array
}

type Script interface {
	Run(tx *Transaction)
}

type Inputs lazyslice.Array
type InputParameters lazyslice.Array
type OutputContexts lazyslice.Array
type Output lazyslice.Array
type TransactionData lazyslice.Array

//- element locator
//- 1 byte transaction level (inps, unlock block, outps, txdata)
//    + local 0xFF byte index level, xFF-1 local byte output level
//- 2 bytes index
//- 1 byte output block index

// script is executed in the context of input. So local context is specified by index

type ElementLocation []byte

type ScriptEmbedded struct {
	LibraryRef byte
	Data       lazyslice.Array
}

func (s *ScriptEmbedded) Run(tx *Transaction) {
	panic("implement me")
}

type ScriptInline struct {
	Code []byte
	Data lazyslice.Array
}

func (s *ScriptInline) Run(tx *Transaction) {
	panic("implement me")
}

type ScriptRef struct {
	ScriptHash     [32]byte
	ScriptLocation []byte
	Data           lazyslice.Array
}

func (s *ScriptRef) Run(eng *engine.Engine, tx *Transaction) {
	eng.Run(nil, tx)
}

func (tx *Transaction) GetElement(elemLocation []byte) ([]byte, bool) {
	panic("implement me")
}
