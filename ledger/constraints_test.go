package ledger

import (
	"crypto/ed25519"
	"math/rand"
	"testing"
	"time"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/lazyslice"
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
		addr := AddressFromED25519PubKey(pubKey)
		out.PutAddressConstraint(addr, ConstraintSigLockED25519)
		outBack, err := OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("output: %d bytes", len(out.Bytes()))

		addrBack, constr := outBack.AddressConstraint()
		require.EqualValues(t, ConstraintSigLockED25519, constr)
		require.EqualValues(t, addr, addrBack)
	})
	t.Run("tokens", func(t *testing.T) {
		out := NewOutput()
		out.PutTokensConstraint(1337)
		outBack, err := OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("output: %d bytes", len(out.Bytes()))

		tokensBack := outBack.Tokens()
		require.EqualValues(t, 1337, tokensBack)
	})
	t.Run("bad tokens", func(t *testing.T) {
		out := NewOutput()
		easyutxo.RequirePanicOrErrorWith(t, func() error {
			out.Tokens()
			return nil
		}, "bounds out of range")
	})
	t.Run("minimum output", func(t *testing.T) {
		out := NewOutput()
		addr := AddressFromED25519PubKey(pubKey)
		out.PutAddressConstraint(addr, ConstraintSigLockED25519)
		out.PutTokensConstraint(1337)
		require.EqualValues(t, 1337, out.Tokens())
		addrBack, _ := out.AddressConstraint()
		require.EqualValues(t, addr, addrBack)
		t.Logf("utxo len %d bytes", len(out.Bytes()))
		require.EqualValues(t, 2, len(out.MasterConstraintList()))
	})
}

func TestConstructTx(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	pubKey, _, err := ed25519.GenerateKey(rnd)
	require.NoError(t, err)

	t.Run("1", func(t *testing.T) {
		glb := NewTransactionContext()
		idx := glb.ConsumeOutput(NewOutput(), DummyOutputID())
		require.EqualValues(t, 0, idx)
	})
	t.Run("1", func(t *testing.T) {
		ctx := NewTransactionContext()
		t.Logf("transaction context bytes 1: %d", len(ctx.Tree().Bytes()))

		out := NewOutput()
		addr := AddressFromED25519PubKey(pubKey)
		out.PutAddressConstraint(addr, ConstraintSigLockED25519)
		out.PutTokensConstraint(1337)
		dummyOid := DummyOutputID()
		idx := ctx.ConsumeOutput(out, dummyOid)
		require.EqualValues(t, 0, idx)
		t.Logf("transaction context bytes 2: %d", len(ctx.Tree().Bytes()))

		idx = ctx.ProduceOutput(out)
		t.Logf("transaction context bytes 3: %d", len(ctx.Tree().Bytes()))
		require.EqualValues(t, 0, idx)

		txbytes := ctx.Transaction().Bytes()
		t.Logf("tx %d bytes", len(txbytes))

		count := 0
		ctx.ForEachProducedOutput(func(out *Output, path lazyslice.TreePath) bool {
			a, c := out.AddressConstraint()
			require.EqualValues(t, a, addr)
			require.EqualValues(t, c, ConstraintSigLockED25519)
			require.EqualValues(t, 1337, out.Tokens())
			count++
			return true
		})
		require.EqualValues(t, 1, count)
		//err = ctx.Validate()
		//require.NoError(t, err)
	})
}
