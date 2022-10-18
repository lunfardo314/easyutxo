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
		outBack := OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("empty output: %d bytes", len(out.Bytes()))
	})
	t.Run("address", func(t *testing.T) {
		out := NewOutput()
		addr := AddressFromED25519PubKey(pubKey)
		out.PutAddress(addr)
		outBack := OutputFromBytes(out.Bytes())
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("output: %d bytes", len(out.Bytes()))

		addrBack := outBack.Address()
		require.EqualValues(t, ConstraintSigLockED25519, addrBack[0])
		require.EqualValues(t, addr, addrBack)
	})
	t.Run("tokens", func(t *testing.T) {
		out := NewOutput()
		out.PutMainConstraint(uint32(time.Now().Unix()), 1337)
		outBack := OutputFromBytes(out.Bytes())
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("output: %d bytes", len(out.Bytes()))

		tokensBack := outBack.Amount()
		require.EqualValues(t, 1337, tokensBack)
	})
	t.Run("bad tokens", func(t *testing.T) {
		out := NewOutput()
		easyutxo.RequirePanicOrErrorWith(t, func() error {
			out.Amount()
			return nil
		}, "bounds out of range")
	})
	t.Run("minimum output", func(t *testing.T) {
		out := NewOutput()
		addr := AddressFromED25519PubKey(pubKey)
		out.PutAddress(addr)
		out.PutMainConstraint(uint32(time.Now().Unix()), 1337)
		require.EqualValues(t, 1337, out.Amount())
		addrBack := out.Address()
		require.EqualValues(t, addr, addrBack)
		t.Logf("utxo len %d bytes", len(out.Bytes()))
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
		out.PutAddress(addr)
		out.PutMainConstraint(uint32(time.Now().Unix()), 1337)
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
		ctx.ForEachOutput(Path(ConsumedContextBranch, ConsumedContextOutputsBranch), func(out *Output, path lazyslice.TreePath) bool {
			a := out.Address()
			require.EqualValues(t, a, addr)
			require.EqualValues(t, a[0], ConstraintSigLockED25519)
			require.EqualValues(t, 1337, out.Amount())
			count++
			return true
		})
		require.EqualValues(t, 1, count)
		//err = ctx.Validate()
		//require.NoError(t, err)
	})
}
