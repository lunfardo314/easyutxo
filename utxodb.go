package easyutxo

import (
	"github.com/lunfardo314/easyutxo/transaction"
)

var newUTXODB func() transaction.LedgerState

func RegisterNewUTXODBConstructor(fun func() transaction.LedgerState) {
	newUTXODB = fun
}

func NewUTXODB() transaction.LedgerState {
	return newUTXODB()
}
