package txbuilder

import (
	"crypto/ed25519"
	"math/rand"
	"testing"
	"time"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger/constraint"
	"github.com/lunfardo314/easyutxo/ledger/state"
	"github.com/lunfardo314/easyutxo/ledger/utxodb"
	"github.com/stretchr/testify/require"
)

func TestOutput(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	pubKey, _, err := ed25519.GenerateKey(rnd)
	require.NoError(t, err)

	const msg = "message to be signed"

	t.Run("basic", func(t *testing.T) {
		out := OutputBasic(0, 0, constraint.AddressED25519Null())
		outBack, err := OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("empty output: %d bytes", len(out.Bytes()))
	})
	t.Run("address", func(t *testing.T) {
		out := OutputBasic(0, 0, constraint.AddressED25519FromPublicKey(pubKey))
		outBack, err := OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("output: %d bytes", len(out.Bytes()))

		_, err = constraint.AddressED25519FromBytes(outBack.Lock())
		require.NoError(t, err)
		require.EqualValues(t, out.Lock(), outBack.Lock())
	})
	t.Run("tokens", func(t *testing.T) {
		out := OutputBasic(1337, uint32(time.Now().Unix()), constraint.AddressED25519Null())
		outBack, err := OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("output: %d bytes", len(out.Bytes()))

		tokensBack := outBack.Amount()
		require.EqualValues(t, 1337, tokensBack)
	})
}

func TestTimelock(t *testing.T) {
	t.Run("time lock 1", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)
		privKey0, _, addr0 := u.GenerateAddress(0)
		err := u.TokensFromFaucet(addr0, 10000)
		require.EqualValues(t, 1, u.NumUTXOs(u.OriginAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.OriginAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 1, u.NumUTXOs(addr0))

		priv1, _, addr1 := u.GenerateAddress(1)

		ts := uint32(time.Now().Unix()) + 5
		par, err := u.MakeED25519TransferInputs(privKey0)
		require.NoError(t, err)
		par.WithAmount(200).
			WithTargetLock(addr1).
			WithTimestamp(ts).
			WithConstraint(constraint.NewTimelock(ts + 1))
		txBytes, err := MakeTransferTransaction(par)

		require.NoError(t, err)
		t.Logf("tx with timelock len: %d", len(txBytes))
		err = u.AddTransaction(txBytes, state.TraceOptionFailedConstraints)
		require.NoError(t, err)

		require.EqualValues(t, 200, u.Balance(addr1))

		par, err = u.MakeED25519TransferInputs(privKey0)
		require.NoError(t, err)
		par.WithAmount(2000).
			WithTargetLock(addr1).
			WithTimestamp(ts + 1).
			WithConstraint(constraint.NewTimelock(ts + 1 + 10))
		err = u.DoTransfer(par)
		require.NoError(t, err)

		require.EqualValues(t, 2200, u.Balance(addr1))

		par, err = u.MakeED25519TransferInputs(priv1)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr0).
			WithTimestamp(ts + 2),
		)

		easyfl.RequireErrorWith(t, err, "constraint 'timelock' failed")
		require.EqualValues(t, 2200, u.Balance(addr1))

		par, err = u.MakeED25519TransferInputs(priv1)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr0).
			WithTimestamp(ts + 12),
		)
		require.NoError(t, err)
		require.EqualValues(t, 200, u.Balance(addr1))
	})
	t.Run("time lock 2", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)

		privKey0, _, addr0 := u.GenerateAddress(0)
		err := u.TokensFromFaucet(addr0, 10000)
		require.EqualValues(t, 1, u.NumUTXOs(u.OriginAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.OriginAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 1, u.NumUTXOs(addr0))

		priv1, _, addr1 := u.GenerateAddress(1)

		ts := uint32(time.Now().Unix()) + 5
		par, err := u.MakeED25519TransferInputs(privKey0)
		require.NoError(t, err)
		txBytes, err := MakeTransferTransaction(par.
			WithAmount(200).
			WithTargetLock(addr1).
			WithTimestamp(ts).
			WithConstraint(constraint.NewTimelock(ts + 1)),
		)
		require.NoError(t, err)
		t.Logf("tx with timelock len: %d", len(txBytes))
		err = u.AddTransaction(txBytes, state.TraceOptionFailedConstraints)
		require.NoError(t, err)

		require.EqualValues(t, 200, u.Balance(addr1))

		par, err = u.MakeED25519TransferInputs(privKey0)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr1).
			WithTimestamp(ts + 1).
			WithConstraint(constraint.NewTimelock(ts + 1 + 10)),
		)
		require.NoError(t, err)

		require.EqualValues(t, 2200, u.Balance(addr1))

		par, err = u.MakeED25519TransferInputs(priv1)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr0).
			WithTimestamp(ts + 2),
		)
		easyfl.RequireErrorWith(t, err, "constraint 'timelock' failed")
		require.EqualValues(t, 2200, u.Balance(addr1))

		par, err = u.MakeED25519TransferInputs(priv1)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr0).
			WithTimestamp(ts + 12),
		)
		require.NoError(t, err)
		require.EqualValues(t, 200, u.Balance(addr1))
	})
}

func TestDeadlineLock(t *testing.T) {
	u := utxodb.NewUTXODB(true)
	privKey0, pubKey0, addr0 := u.GenerateAddress(0)
	err := u.TokensFromFaucet(addr0, 10000)
	require.NoError(t, err)
	require.EqualValues(t, 1, u.NumUTXOs(u.OriginAddress()))
	require.EqualValues(t, u.Supply()-10000, u.Balance(u.OriginAddress()))
	require.EqualValues(t, 10000, u.Balance(addr0))
	require.EqualValues(t, 1, u.NumUTXOs(addr0))

	privKey1, pubKey1, addr1 := u.GenerateAddress(1)
	require.EqualValues(t, 0, u.Balance(addr1))
	require.EqualValues(t, 0, u.NumUTXOs(addr1))
	privKey1 = privKey1

	ts := uint32(time.Now().Unix())

	par, err := u.MakeED25519TransferInputs(privKey0)
	require.NoError(t, err)
	deadlineLock := constraint.NewDeadlineLock(
		ts+10,
		constraint.AddressED25519FromPublicKey(pubKey1),
		constraint.AddressED25519FromPublicKey(pubKey0),
	)
	t.Logf("deadline lock: %d bytes", len(deadlineLock.Bytes()))
	txBytes, err := u.DoTransferTx(par.
		WithAmount(2000).
		WithTargetLock(deadlineLock).
		WithTimestamp(ts),
	)
	require.NoError(t, err)
	require.EqualValues(t, 10000-2000, u.Balance(addr0))
	t.Logf("tx ith deadline lock: %d bytes", len(txBytes))
	ctx, err := state.ValidationContextFromTransaction(txBytes, u)
	require.NoError(t, err)
	require.EqualValues(t, 2, ctx.NumProducedOutputs())
}
