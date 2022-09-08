package opcodes_test

import (
	"testing"

	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/opcodes"
	"github.com/lunfardo314/easyutxo/ledger/utxodb"
	"github.com/stretchr/testify/require"
)

func TestOpcodes(t *testing.T) {
	//t.Run("opcodes1", func(t *testing.T) {
	//	require.True(t, opcodes.OpsExit.IsShort())
	//	require.True(t, opcodes.OpsVerifySigED25519.IsShort())
	//	require.False(t, opcodes.OplReserved126.IsShort())
	//	oc := opcodes.OpCode(0)
	//	t.Logf("%s", oc)
	//	oc = opcodes.OpsExit
	//	t.Logf("%s", oc)
	//	oc = opcodes.OpsVerifySigED25519
	//	t.Logf("%s", oc)
	//	oc = opcodes.OpCode(31)
	//	t.Logf("%s", oc)
	//	oc = opcodes.OplReserved126
	//	t.Logf("%s", oc)
	//})
	t.Run("opcodes2", func(t *testing.T) {
		d := []byte{200}
		require.Panics(t, func() {
			opcodes.ParseOpcode(d)
		})
		d = []byte{0x01, 0x02}
		require.NotPanics(t, func() {
			opcodes.ParseOpcode(d)
		})
	})

}

func TestBasic(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		tx := ledger.New()
		v, err := tx.CreateGlobalContext(utxodb.New())
		require.NoError(t, err)
		engine.Run(opcodes.All, v.Tree(), nil, 0, nil, nil)
	})

}
