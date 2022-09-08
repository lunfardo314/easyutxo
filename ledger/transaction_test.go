package ledger_test

import (
	"testing"

	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/utxodb"
	"github.com/stretchr/testify/require"
)

func TestBasics(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		tx := ledger.New()
		t.Logf("empty tx size: %d", len(tx.Bytes()))
	})
	t.Run("2", func(t *testing.T) {
		utxodb := utxodb.New()
		tx := ledger.New()
		require.EqualValues(t, 0, tx.NumInputs())
		require.EqualValues(t, 0, tx.NumOutputs())
		v, err := ledger.CreateGlobalContext(tx, utxodb)
		require.NoError(t, err)
		txBack := v.Transaction()
		require.EqualValues(t, tx.Bytes(), txBack.Bytes())
		v.ValidateOutputs()
	})

}
