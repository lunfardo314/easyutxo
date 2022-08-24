package transaction

import "testing"

func TestBasics(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		tx := NewTransaction()
		t.Logf("empty tx size: %d", len(tx.Bytes()))
	})
}
