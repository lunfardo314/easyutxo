package ledger

import (
	"crypto/ed25519"
	"math/rand"
	"testing"
	"time"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger/constraint"
	"github.com/stretchr/testify/require"
)

func TestOutput(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	pubKey, _, err := ed25519.GenerateKey(rnd)
	require.NoError(t, err)

	const msg = "message to be signed"

	t.Run("basic", func(t *testing.T) {
		out := OutputBasic(0, 0, constraint.AddressED25519SigLockNull())
		outBack, err := OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("empty output: %d bytes", len(out.Bytes()))
	})
	t.Run("address", func(t *testing.T) {
		out := OutputBasic(0, 0, constraint.AddressED25519SigLock(pubKey))
		outBack, err := OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("output: %d bytes", len(out.Bytes()))

		_, ok := constraint.ParseAddressED25519Constraint(outBack.Lock())
		require.True(t, ok)
		require.EqualValues(t, out.Lock(), outBack.Lock())
	})
	t.Run("tokens", func(t *testing.T) {
		out := OutputBasic(1337, uint32(time.Now().Unix()), constraint.AddressED25519SigLockNull())
		outBack, err := OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("output: %d bytes", len(out.Bytes()))

		tokensBack := outBack.Amount()
		require.EqualValues(t, 1337, tokensBack)
	})
}

func TestConstructTx(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	pubKey, _, err := ed25519.GenerateKey(rnd)
	require.NoError(t, err)

	t.Run("1", func(t *testing.T) {
		glb := NewTransactionContext()
		idx, err := glb.ConsumeOutput(OutputBasic(0, 0, nil), DummyOutputID())
		require.NoError(t, err)
		require.EqualValues(t, 0, idx)
	})
	t.Run("1", func(t *testing.T) {
		ctx := NewTransactionContext()
		t.Logf("transaction bytes 1: %d", len(ctx.Transaction.Bytes()))

		out := OutputBasic(1337, uint32(time.Now().Unix()), constraint.AddressED25519SigLock(pubKey))
		dummyOid := DummyOutputID()
		idx, err := ctx.ConsumeOutput(out, dummyOid)
		require.NoError(t, err)
		require.EqualValues(t, 0, idx)
		t.Logf("transaction bytes 2: %d", len(ctx.Transaction.Bytes()))

		idx, err = ctx.ProduceOutput(out)
		require.NoError(t, err)
		t.Logf("transaction bytes 3: %d", len(ctx.Transaction.Bytes()))
		require.EqualValues(t, 0, idx)

		txbytes := ctx.Transaction.Bytes()
		t.Logf("tx %d bytes", len(txbytes))
	})
}

func TestConstraintAmount(t *testing.T) {
	t.Run("base", func(t *testing.T) {
		src := "validAmount(u64/1)"
		res, err := easyfl.EvalFromSource(easyfl.NewGlobalDataTracePrint(nil), src)
		require.NoError(t, err)
		require.True(t, len(res) > 0)
	})
	t.Run("code", func(t *testing.T) {
		src := "validAmount(u64/1)"
		_, numArgs, binCode, err := easyfl.CompileExpression(src)
		require.NoError(t, err)
		require.EqualValues(t, 0, numArgs)
		t.Logf("%s: bin code: %s", src, easyfl.Fmt(binCode))
	})
}
