package utxodb

import (
	"bytes"

	"github.com/lunfardo314/easyutxo/ledger"
)

type UtxoDB struct {
	utxo map[string]*ledger.Output
}

func New() *UtxoDB {
	return &UtxoDB{
		utxo: make(map[string]*ledger.Output),
	}
}

func (u *UtxoDB) AddTransaction(txBytes []byte) error {
	ctx, err := ledger.GlobalContextFromTransaction(txBytes, u)
	if err != nil {
		return err
	}
	if err = ctx.Validate(); err != nil {
		return err
	}
	// remove spent outputs
	tx := ctx.Transaction()
	tx.ForEachInputID(func(_ uint16, o ledger.OutputID) bool {
		delete(u.utxo, string(o[:]))
		return true
	})
	// add new outputs
	txid := tx.ID()
	tx.ForEachOutput(func(group, idx byte, o *ledger.Output) bool {
		id := ledger.NewOutputID(txid, group, idx)
		u.utxo[string(id[:])] = o
		return true
	})
	return nil
}

func (u *UtxoDB) GetUTXO(id *ledger.OutputID) (ledger.OutputData, bool) {
	ret, ok := u.utxo[string(id[:])]
	return ret.Bytes(), ok
}

func (u *UtxoDB) GetUTXOsForAddress(addr []byte) []ledger.OutputData {
	ret := make([]ledger.OutputData, 0)
	for _, d := range u.utxo {
		if bytes.Equal(addr, d.Address()) {
			ret = append(ret, d.Bytes())
		}
	}
	return ret
}
