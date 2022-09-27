package ledger

import (
	"crypto/ed25519"
	"math/rand"
	"testing"
	"time"

	"github.com/lunfardo314/easyutxo"
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
		out.PutTokens(1337)
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
		}, "must be 8 bytes")
	})
}
