package transaction

import (
	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

const (
	TransactionIDLength = 32
	OutputIDLength      = TransactionIDLength + 2
)

type Transaction struct {
	tree *lazyslice.Tree
}

func New() *Transaction {
	ret := &Transaction{tree: lazyslice.TreeEmpty()}
	ret.tree.PushNewSubtreeAtPath() // input groups
	ret.tree.PushNewSubtreeAtPath() // param groups
	ret.tree.PushNewSubtreeAtPath() // output contexts
	ret.tree.PushNewSubtreeAtPath() // transaction level data elements
	return ret
}

func FromBytes(data []byte) *Transaction {
	return &Transaction{tree: lazyslice.TreeFromBytes(data)}
}

func (tx *Transaction) Bytes() []byte {
	return tx.tree.Bytes()
}

func (tx *Transaction) Validate() error {
	return easyutxo.CatchPanic(func() {
		tx.RunValidationScripts()
	})

}

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

func (tx *Transaction) RunValidationScripts() {

}

func (tx *Transaction) CheckUnboundedConstraints() {

}
