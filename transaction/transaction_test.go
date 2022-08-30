package transaction

import (
	"testing"

	"github.com/lunfardo314/easyutxo"
	"github.com/stretchr/testify/require"
)

func TestBasics(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		tx := New()
		t.Logf("empty tx size: %d", len(tx.Bytes()))
	})
	t.Run("2", func(t *testing.T) {
		ledger := easyutxo.NewUTXODB()
		tx := New()
		v, err := tx.GetValidationContext(ledger)
		require.NoError(t, err)
		txBack := v.Transaction()
		require.EqualValues(t, tx.Bytes(), txBack.Bytes())
	})

}
