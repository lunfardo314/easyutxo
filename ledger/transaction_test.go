package ledger_test

import (
	"bytes"
	"testing"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/stretchr/testify/require"
)

func TestBasics(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		tx := ledger.NewTransaction()
		t.Logf("empty tx size: %d", len(tx.Bytes()))
		require.EqualValues(t, 0, tx.NumInputs())
		require.EqualValues(t, 0, tx.NumOutputs())
	})
	t.Run("2", func(t *testing.T) {
		utxodb := ledger.NewUTXODBInMemory()
		tx := ledger.NewTransaction()
		require.EqualValues(t, 0, tx.NumInputs())
		require.EqualValues(t, 0, tx.NumOutputs())

		v, err := ledger.TransactionContextFromTransaction(tx.Bytes(), utxodb)
		require.NoError(t, err)
		txid := tx.ID()
		require.EqualValues(t, txid, v.TransactionID())

		inputIDs := v.MustEval("txInputIDsBytes", nil)
		v2 := tx.Tree().BytesAtPath(ledger.Path(ledger.TxInputIDsBranch))
		require.EqualValues(t, inputIDs, v2)

		outputs := v.MustEval("txOutputsBytes", nil)
		v2 = tx.Tree().BytesAtPath(ledger.Path(ledger.TxOutputBranch))
		require.EqualValues(t, outputs, v2)

		ts := v.MustEval("txTimestampBytes", nil)
		v2 = tx.Tree().BytesAtPath(ledger.Path(ledger.TxTimestamp))
		require.True(t, easyutxo.EmptySlices(ts, v2))

		inpComm := v.MustEval("txInputCommitmentBytes", nil)
		v2 = tx.Tree().BytesAtPath(ledger.Path(ledger.TxInputCommitment))
		require.True(t, easyutxo.EmptySlices(inpComm, v2))

		lib := v.MustEval("txLocalLibBytes", nil)
		v2 = tx.Tree().BytesAtPath(ledger.Path(ledger.TxLocalLibraryBranch))
		require.True(t, bytes.Equal(lib, v2))

		essence := v.MustEval("txEssenceBytes", nil)
		v2 = easyutxo.Concat(inputIDs, outputs, ts, inpComm, lib)
		require.True(t, bytes.Equal(essence, v2))

	})
}
