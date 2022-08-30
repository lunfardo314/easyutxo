package utxodb

import (
	"bytes"

	"github.com/lunfardo314/easyutxo/transaction"
)

type UtxoDB struct {
	utxo map[string]*transaction.Output
}

func New() *UtxoDB {
	return &UtxoDB{
		utxo: make(map[string]*transaction.Output),
	}
}

func (u *UtxoDB) AddTransaction(tx *transaction.Transaction) error {
	_, err := tx.GetValidationContext(u)
	if err != nil {
		return err
	}
	// TODO run validation scripts
	// remove spent outputs
	tx.ForEachInput(func(idx uint16, o transaction.OutputID) bool {
		delete(u.utxo, string(o[:]))
		return true
	})
	// add new outputs
	txid := tx.ID()
	tx.ForEachOutput(func(idx uint16, o *transaction.Output) bool {
		id := transaction.NewOutputID(txid, idx)
		u.utxo[string(id[:])] = o
		return true
	})
	return nil
}

func (u *UtxoDB) GetUTXO(id *transaction.OutputID) (transaction.OutputData, bool) {
	ret, ok := u.utxo[string(id[:])]
	return ret.Bytes(), ok
}

func (u *UtxoDB) GetUTXOsForAddress(addr []byte) []transaction.OutputData {
	ret := make([]transaction.OutputData, 0)
	for _, d := range u.utxo {
		if bytes.Equal(addr, d.Address()) {
			ret = append(ret, d.Bytes())
		}
	}
	return ret
}
