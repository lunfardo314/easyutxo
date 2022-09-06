package opcodes

import (
	"strings"
	"testing"

	"github.com/lunfardo314/easyutxo/engine"
	"github.com/stretchr/testify/require"
)

func TestAssembly(t *testing.T) {
	t.Run("empty program", func(t *testing.T) {
		code, err := GenProgram(func(p *engine.Program) {
		})
		require.NoError(t, err)
		t.Logf("code len: %d", len(code))
	})
	t.Run("first program", func(t *testing.T) {
		code, err := GenProgram(func(p *engine.Program) {
			p.OP(OpsNOP)
			p.OP(OpsExit)
			p.OP(OpsSigLockED25519)
			p.OP(OplReserved126)
		})
		require.NoError(t, err)
		t.Logf("code len: %d", len(code))
	})
	t.Run("with dummy label", func(t *testing.T) {
		code, err := GenProgram(func(p *engine.Program) {
			p.OP(OpsNOP)
			p.OP(OpsExit)
			p.Label("dummy")
			p.OP(OpsSigLockED25519)
			p.OP(OplReserved126)
		})
		require.NoError(t, err)
		t.Logf("code len: %d", len(code))
	})
	t.Run("wrong instruction", func(t *testing.T) {
		_, err := GenProgram(func(p *engine.Program) {
			p.OP(OpsNOP)
			p.OP(OpCode(100))
		})
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "wrong opcode"))
	})
}
