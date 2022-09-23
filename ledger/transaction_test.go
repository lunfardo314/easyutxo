package ledger_test

import (
	"bytes"
	"testing"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/globalpath"
	"github.com/lunfardo314/easyutxo/ledger/utxodb"
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
		utxodb := utxodb.New()
		tx := ledger.NewTransaction()
		require.EqualValues(t, 0, tx.NumInputs())
		require.EqualValues(t, 0, tx.NumOutputs())

		v, err := ledger.GlobalContextFromTransaction(tx.Bytes(), utxodb)
		require.NoError(t, err)
		txid := tx.ID()
		require.EqualValues(t, txid, v.TransactionID())

		inputIDs := v.Eval("txInputIDsBytes", nil)
		v2 := tx.Tree().BytesAtPath(ledger.Path(globalpath.TxInputIDsLongIndex))
		require.EqualValues(t, inputIDs, v2)

		outputs := v.Eval("txOutputBytes", nil)
		v2 = tx.Tree().BytesAtPath(ledger.Path(globalpath.TxOutputGroupsIndex))
		require.EqualValues(t, outputs, v2)

		ts := v.Eval("txTimestampBytes", nil)
		v2 = tx.Tree().BytesAtPath(ledger.Path(globalpath.TxTimestampIndex))
		require.True(t, easyutxo.EmptySlices(ts, v2))

		inpComm := v.Eval("txInputCommitmentBytes", nil)
		v2 = tx.Tree().BytesAtPath(ledger.Path(globalpath.TxInputCommitmentIndex))
		require.True(t, easyutxo.EmptySlices(inpComm, v2))

		lib := v.Eval("txLocalLibBytes", nil)
		v2 = tx.Tree().BytesAtPath(ledger.Path(globalpath.TxLocalLibraryIndex))
		require.True(t, bytes.Equal(lib, v2))

		essence := v.Eval("txEssenceBytes", nil)
		v2 = easyutxo.Concat(inputIDs, outputs, ts, inpComm, lib)
		require.True(t, bytes.Equal(essence, v2))
	})

}
