package ledger

import (
	"crypto/ed25519"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOutput(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	pubKey, _, err := ed25519.GenerateKey(rnd)
	require.NoError(t, err)

	const msg = "message to be signed"

	t.Run("basic", func(t *testing.T) {
		out := NewOutput()
		outBack, err := OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("empty output: %d bytes", len(out.Bytes()))
	})
	t.Run("address", func(t *testing.T) {
		out := NewOutput()
		out.Address = AddressFromED25519PubKey(pubKey)
		outBack, err := OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("output: %d bytes", len(out.Bytes()))

		require.EqualValues(t, ConstraintSigLockED25519, outBack.Address[0])
		require.EqualValues(t, out.Address, outBack.Address)
	})
	t.Run("tokens", func(t *testing.T) {
		out := NewOutput()
		out.Timestamp = uint32(time.Now().Unix())
		out.Amount = 1337
		outBack, err := OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("output: %d bytes", len(out.Bytes()))

		tokensBack := outBack.Amount
		require.EqualValues(t, 1337, tokensBack)
	})
}

func TestConstructTx(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	pubKey, _, err := ed25519.GenerateKey(rnd)
	require.NoError(t, err)

	t.Run("1", func(t *testing.T) {
		glb := NewTransactionContext()
		idx, err := glb.ConsumeOutput(NewOutput(), DummyOutputID())
		require.NoError(t, err)
		require.EqualValues(t, 0, idx)
	})
	t.Run("1", func(t *testing.T) {
		ctx := NewTransactionContext()
		t.Logf("transaction bytes 1: %d", len(ctx.Transaction.Bytes()))

		out := NewOutput()
		out.Address = AddressFromED25519PubKey(pubKey)
		out.Timestamp = uint32(time.Now().Unix())
		out.Amount = 1337
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
