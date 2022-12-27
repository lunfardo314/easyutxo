package utxodb

import (
	"testing"
)

func TestGenesis(t *testing.T) {
	u := NewUTXODB()
	t.Logf("\n%s", u.GenesisData())
}
