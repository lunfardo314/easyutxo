package utxodb

import (
	"bytes"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyutxo/ledger"
)

type UtxoDB struct {
	store common.KVStore
}

func New(store common.KVStore) *UtxoDB {
	return &UtxoDB{
		store: store,
	}
}

func NewInMemory() *UtxoDB {
	return New(common.NewInMemoryKVStore())
}

func (u *UtxoDB) utxoPartition() common.KVStore {
	return u.store
}

func (u *UtxoDB) AddTransaction(txBytes []byte) error {
	ctx, err := ledger.TransactionContextFromTransaction(txBytes, u)
	if err != nil {
		return err
	}
	if err = ctx.Validate(); err != nil {
		return err
	}
	u.updateLedger(ctx.Transaction())
	return nil
}

func (u *UtxoDB) GetUTXO(id *ledger.OutputID) (ledger.OutputData, bool) {
	ret := u.utxoPartition().Get(id.Bytes())
	if len(ret) == 0 {
		return nil, false
	}
	return ret, true
}

func (u *UtxoDB) GetUTXOsForAddress(addr []byte) []ledger.OutputData {
	ret := make([]ledger.OutputData, 0)
	u.utxoPartition().Iterate(func(k, v []byte) bool {
		o, err := ledger.OutputFromBytes(v)
		common.AssertNoError(err)
		a, constraint := o.AddressConstraint()
		common.Assert(constraint == ledger.ConstraintSigLockED25519, "only ConstraintSigLockED25519 supported")
		if bytes.Equal(addr, a) {
			ret = append(ret, v)
		}
		return true
	})
	return ret
}

// updateLedger in the future must be atomic
func (u *UtxoDB) updateLedger(tx *ledger.Transaction) {
	tx.ForEachInputID(func(_ byte, o ledger.OutputID) bool {
		u.utxoPartition().Set(o[:], nil)
		return true
	})
	// add new outputs
	txid := tx.ID()
	tx.ForEachOutput(func(o *ledger.Output, idx byte) bool {
		id := ledger.NewOutputID(txid, idx)
		u.utxoPartition().Set(id[:], o.Bytes())
		return true
	})
}
