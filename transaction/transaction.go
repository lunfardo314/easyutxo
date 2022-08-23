package transaction

import (
	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/engine"
)

const (
	TransactionIDLength = 32
	OutputIDLength      = TransactionIDLength + 2
)

type OutputBlock struct {
	Script Script
	Params easyutxo.SliceArray
}

type Script interface {
	Run(tx *Transaction)
}

type Transaction struct {
	easyutxo.SliceArray
	Inputs          Inputs
	InputParameters InputParameters
	OutputContexts  OutputContexts
	TransactionData TransactionData
}

type Inputs easyutxo.SliceArray
type InputParameters easyutxo.SliceArray
type OutputContexts easyutxo.SliceArray
type Output easyutxo.SliceArray
type TransactionData easyutxo.SliceArray

func TransactionFromBytes(txbytes []byte) *Transaction {
	ret := &Transaction{
		SliceArray: *easyutxo.SliceArrayFromBytes(txbytes),
	}
	ret.Inputs = InputsFromBytes(ret.At(0))
	return ret
}

func InputsFromBytes(inputsBytes []byte) Inputs {
	return Inputs(*easyutxo.SliceArrayFromBytes(inputsBytes))
}

//- element locator
//- 1 byte transaction level (inps, unlock block, outps, txdata)
//    + local 0xFF byte index level, xFF-1 local byte output level
//- 2 bytes index
//- 1 byte output block index

// script is executed in the context of input. So local context is specified by index

type ElementLocation []byte

type ScriptEmbedded struct {
	LibraryRef byte
	Data       easyutxo.SliceArray
}

func (s *ScriptEmbedded) Run(tx *Transaction) {
	panic("implement me")
}

type ScriptInline struct {
	Code []byte
	Data easyutxo.SliceArray
}

func (s *ScriptInline) Run(tx *Transaction) {
	panic("implement me")
}

type ScriptRef struct {
	ScriptHash     [32]byte
	ScriptLocation []byte
	Data           easyutxo.SliceArray
}

func (s *ScriptRef) Run(eng *engine.Engine, tx *Transaction) {
	eng.Run(nil, tx)
}

func (tx *Transaction) GetElement(elemLocation []byte) ([]byte, bool) {
	panic("implement me")
}
