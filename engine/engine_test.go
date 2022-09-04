package engine_test

import (
	"testing"

	"github.com/lunfardo314/easyutxo/engine"
	"github.com/lunfardo314/easyutxo/transaction"
	"github.com/lunfardo314/easyutxo/utxodb"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	t.Run("opcode", func(t *testing.T) {
		t.Logf("%s", engine.OpCode(0).String())
		t.Logf("%s", engine.OP_EXIT)
		t.Logf("%s", engine.OpCode(31))
		d := []byte{engine.ExtendedOpcodeMask | 0x01}
		require.Panics(t, func() {
			engine.ParseOpcode(d)
		})
		d = []byte{0x01, 0x02}
		require.NotPanics(t, func() {
			engine.ParseOpcode(d)
		})
	})
	t.Run("1", func(t *testing.T) {
		tx := transaction.New()
		v, err := tx.CreateValidationContext(utxodb.New())
		require.NoError(t, err)
		engine.Run(v.Tree(), nil, nil, nil)
	})

}
