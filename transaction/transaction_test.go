package transaction_test

import (
	"testing"

	"github.com/lunfardo314/easyutxo/transaction"
	"github.com/lunfardo314/easyutxo/utxodb"
	"github.com/stretchr/testify/require"
)

func TestBasics(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		tx := transaction.New()
		t.Logf("empty tx size: %d", len(tx.Bytes()))
	})
	t.Run("2", func(t *testing.T) {
		ledger := utxodb.New()
		tx := transaction.New()
		v, err := tx.GetValidationContext(ledger)
		require.NoError(t, err)
		txBack := v.Transaction()
		require.EqualValues(t, tx.Bytes(), txBack.Bytes())
		v.ValidateOutputs()
	})

}
