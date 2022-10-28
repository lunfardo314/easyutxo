package ledger_test

import (
	"testing"
	"time"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/stretchr/testify/require"
)

func TestBasics(t *testing.T) {
	t.Run("empty tx", func(t *testing.T) {
		ctx := ledger.NewTransactionContext()
		t.Logf("empty ctx size: %d", len(ctx.Transaction.Bytes()))
		require.EqualValues(t, 0, ctx.NumInputs())
		require.EqualValues(t, 0, ctx.NumOutputs())
	})
	//t.Run("tx structure", func(t *testing.T) {
	//	state := ledger.NewLedgerStateInMemory(nil, 100)
	//	tx := ledger.NewTransaction()
	//	require.EqualValues(t, 0, tx.NumInputs())
	//	require.EqualValues(t, 0, tx.NumOutputs())
	//
	//	v, err := ledger.ValidationContextFromTransaction(tx.Bytes(), state)
	//	require.NoError(t, err)
	//	txid := tx.ID()
	//	require.EqualValues(t, txid, v.TransactionID())
	//
	//	inputIDs := v.MustEval("txInputIDsBytes", nil)
	//	v2 := tx.Tree().BytesAtPath(ledger.Path(ledger.TxInputIDsBranch))
	//	require.EqualValues(t, inputIDs, v2)
	//
	//	outputs := v.MustEval("txOutputsBytes", nil)
	//	v2 = tx.Tree().BytesAtPath(ledger.Path(ledger.TxOutputBranch))
	//	require.EqualValues(t, outputs, v2)
	//
	//	ts := v.MustEval("txTimestampBytes", nil)
	//	v2 = tx.Tree().BytesAtPath(ledger.Path(ledger.TxTimestamp))
	//	require.True(t, easyutxo.EmptySlices(ts, v2))
	//
	//	inpComm := v.MustEval("txInputCommitmentBytes", nil)
	//	v2 = tx.Tree().BytesAtPath(ledger.Path(ledger.TxInputCommitment))
	//	require.True(t, easyutxo.EmptySlices(inpComm, v2))
	//
	//	lib := v.MustEval("txLocalLibBytes", nil)
	//	v2 = tx.Tree().BytesAtPath(ledger.Path(ledger.TxLocalLibraryBranch))
	//	require.True(t, bytes.Equal(lib, v2))
	//
	//	essence := v.MustEval("txEssenceBytes", nil)
	//	v2 = easyutxo.Concat(inputIDs, outputs, ts, inpComm, lib)
	//	require.True(t, bytes.Equal(essence, v2))
	//
	//})
	//t.Run("input commitment", func(t *testing.T) {
	//	ctx := ledger.NewValidationContext()
	//	ctx.AddInputCommitment()
	//	ic := ctx.InputCommitment()
	//	ic1 := ctx.Tree().BytesAtPath(ledger.Path(ledger.TransactionBranch, ledger.TxInputCommitment))
	//	require.EqualValues(t, ic, ic1)
	//	t.Logf("input commitment: %s", hex.EncodeToString(ic))
	//})
	t.Run("utxodb 1", func(t *testing.T) {
		u := ledger.NewUTXODB(true)
		priv, pub := u.OriginKeys()
		t.Logf("orig priv key: %s", easyfl.Fmt(priv))
		t.Logf("orig pub key: %s", easyfl.Fmt(pub))
		t.Logf("origin address: %s", easyfl.Fmt(u.OriginAddress()))

		_, _, addr := u.GenerateAddress(0)
		err := u.TokensFromFaucet(addr, 100)
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.OriginAddress()))
		require.EqualValues(t, u.Supply()-100, u.Balance(u.OriginAddress()))
		require.EqualValues(t, 100, u.Balance(addr))
		require.EqualValues(t, 1, u.NumUTXOs(addr))
	})
	t.Run("utxodb 2", func(t *testing.T) {
		u := ledger.NewUTXODB(true)
		priv, pub := u.OriginKeys()
		t.Logf("orig priv key: %s", easyfl.Fmt(priv))
		t.Logf("orig pub key: %s", easyfl.Fmt(pub))
		t.Logf("origin address: %s", easyfl.Fmt(u.OriginAddress()))

		privKey, _, addr := u.GenerateAddress(0)
		err := u.TokensFromFaucet(addr, 100)
		require.NoError(t, err)
		err = u.TokensFromFaucet(addr)
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.OriginAddress()))
		require.EqualValues(t, u.Supply()-100-ledger.TokensFromFaucetDefault, u.Balance(u.OriginAddress()))
		require.EqualValues(t, 100+ledger.TokensFromFaucetDefault, u.Balance(addr))
		require.EqualValues(t, 2, u.NumUTXOs(addr))

		err = u.TransferTokens(privKey, addr, u.Balance(addr))
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.OriginAddress()))
		require.EqualValues(t, u.Supply()-100-ledger.TokensFromFaucetDefault, u.Balance(u.OriginAddress()))
		require.EqualValues(t, 100+ledger.TokensFromFaucetDefault, u.Balance(addr))
		require.EqualValues(t, 1, u.NumUTXOs(addr))
	})
	t.Run("utxodb 3 compress outputs", func(t *testing.T) {
		u := ledger.NewUTXODB(true)
		priv, pub := u.OriginKeys()
		t.Logf("orig priv key: %s", easyfl.Fmt(priv))
		t.Logf("orig pub key: %s", easyfl.Fmt(pub))
		t.Logf("origin address: %s", easyfl.Fmt(u.OriginAddress()))

		privKey, _, addr := u.GenerateAddress(0)
		const howMany = 256

		total := uint64(0)
		numOuts := 0
		for i := uint64(100); i <= howMany; i++ {
			err := u.TokensFromFaucet(addr, i)
			require.NoError(t, err)
			total += i
			numOuts++

			require.EqualValues(t, 1, u.NumUTXOs(u.OriginAddress()))
			require.EqualValues(t, u.Supply()-total, u.Balance(u.OriginAddress()))
			require.EqualValues(t, total, u.Balance(addr))
			require.EqualValues(t, numOuts, u.NumUTXOs(addr))
		}

		txBytes, err := ledger.MakeTransferTransaction(u, ledger.TransferTransactionParams{
			SenderKey:     privKey,
			TargetAddress: addr,
			Amount:        u.Balance(addr),
		})
		require.NoError(t, err)
		t.Logf("tx size = %d bytes", len(txBytes))

		err = u.TransferTokens(privKey, addr, u.Balance(addr))
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.OriginAddress()))
		require.EqualValues(t, u.Supply()-total, u.Balance(u.OriginAddress()))
		require.EqualValues(t, total, u.Balance(addr))
		require.EqualValues(t, 1, u.NumUTXOs(addr))
	})
	t.Run("utxodb too many inputs", func(t *testing.T) {
		u := ledger.NewUTXODB(true)
		priv, pub := u.OriginKeys()
		t.Logf("orig priv key: %s", easyfl.Fmt(priv))
		t.Logf("orig pub key: %s", easyfl.Fmt(pub))
		t.Logf("origin address: %s", easyfl.Fmt(u.OriginAddress()))

		privKey, _, addr := u.GenerateAddress(0)
		const howMany = 500

		total := uint64(0)
		numOuts := 0
		for i := uint64(100); i <= howMany; i++ {
			err := u.TokensFromFaucet(addr, i)
			require.NoError(t, err)
			total += i
			numOuts++

			require.EqualValues(t, 1, u.NumUTXOs(u.OriginAddress()))
			require.EqualValues(t, u.Supply()-total, u.Balance(u.OriginAddress()))
			require.EqualValues(t, total, u.Balance(addr))
			require.EqualValues(t, numOuts, u.NumUTXOs(addr))
		}
		err := u.TransferTokens(privKey, addr, u.Balance(addr))
		easyfl.RequireErrorWith(t, err, "exceeded max number of consumed outputs")
	})
	t.Run("utxodb fan out outputs", func(t *testing.T) {
		u := ledger.NewUTXODB(true)
		priv, pub := u.OriginKeys()
		t.Logf("orig priv key: %s", easyfl.Fmt(priv))
		t.Logf("orig pub key: %s", easyfl.Fmt(pub))
		t.Logf("origin address: %s", easyfl.Fmt(u.OriginAddress()))

		privKey0, _, addr0 := u.GenerateAddress(0)
		const howMany = 100
		err := u.TokensFromFaucet(addr0, howMany*100)
		require.EqualValues(t, 1, u.NumUTXOs(u.OriginAddress()))
		require.EqualValues(t, u.Supply()-howMany*100, u.Balance(u.OriginAddress()))
		require.EqualValues(t, howMany*100, int(u.Balance(addr0)))
		require.EqualValues(t, 1, u.NumUTXOs(addr0))

		privKey1, _, addr1 := u.GenerateAddress(1)

		for i := 0; i < howMany; i++ {
			err = u.TransferTokens(privKey0, addr1, 100)
			require.NoError(t, err)
		}
		require.EqualValues(t, howMany*100, int(u.Balance(addr1)))
		require.EqualValues(t, howMany, u.NumUTXOs(addr1))
		require.EqualValues(t, 0, u.Balance(addr0))
		require.EqualValues(t, 0, u.NumUTXOs(addr0))

		outs, err := u.GetUTXOsForAddress(addr1)
		require.NoError(t, err)
		require.EqualValues(t, howMany, len(outs))
		for _, o := range outs {
			_, ok := o.Output.Sender()
			require.False(t, ok)
		}

		err = u.TransferTokens(privKey1, addr0, howMany*100, true)
		require.EqualValues(t, howMany*100, u.Balance(addr0))
		require.EqualValues(t, 1, u.NumUTXOs(addr0))
		require.EqualValues(t, 0, u.Balance(addr1))
		require.EqualValues(t, 0, u.NumUTXOs(addr1))

		outs, err = u.GetUTXOsForAddress(addr0)
		require.NoError(t, err)
		require.EqualValues(t, 1, len(outs))

		snd, ok := outs[0].Output.Sender()
		require.True(t, ok)
		require.EqualValues(t, addr1, snd)
	})
	t.Run("time lock 1", func(t *testing.T) {
		u := ledger.NewUTXODB(true)
		priv, pub := u.OriginKeys()
		t.Logf("orig priv key: %s", easyfl.Fmt(priv))
		t.Logf("orig pub key: %s", easyfl.Fmt(pub))
		t.Logf("origin address: %s", easyfl.Fmt(u.OriginAddress()))

		privKey0, _, addr0 := u.GenerateAddress(0)
		err := u.TokensFromFaucet(addr0, 10000)
		require.EqualValues(t, 1, u.NumUTXOs(u.OriginAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.OriginAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 1, u.NumUTXOs(addr0))

		priv1, _, addr1 := u.GenerateAddress(1)

		ts := uint32(time.Now().Unix()) + 5
		txBytes, err := ledger.MakeTransferTransaction(u, ledger.TransferTransactionParams{
			SenderKey:         privKey0,
			TargetAddress:     addr1,
			Amount:            200,
			Timestamp:         ts,
			AddTimeLockForSec: 1,
		})
		require.NoError(t, err)
		t.Logf("tx with timelock len: %d", len(txBytes))
		err = u.AddTransaction(txBytes, ledger.TraceOptionFailedConstraints)
		require.NoError(t, err)

		require.EqualValues(t, 200, u.Balance(addr1))

		err = u.DoTransfer(ledger.TransferTransactionParams{
			SenderKey:         privKey0,
			TargetAddress:     addr1,
			Amount:            2000,
			AddTimeLockForSec: 10,
			Timestamp:         ts + 1,
		})
		require.NoError(t, err)

		require.EqualValues(t, 2200, u.Balance(addr1))
		err = u.DoTransfer(ledger.TransferTransactionParams{
			SenderKey:     priv1,
			TargetAddress: addr0,
			Amount:        2000,
			Timestamp:     ts + 2,
		})
		easyfl.RequireErrorWith(t, err, "constraint 'timelock' failed")
		require.EqualValues(t, 2200, u.Balance(addr1))

		err = u.DoTransfer(ledger.TransferTransactionParams{
			SenderKey:     priv1,
			TargetAddress: addr0,
			Amount:        2000,
			Timestamp:     ts + 12,
		})
		require.NoError(t, err)
		require.EqualValues(t, 200, u.Balance(addr1))
	})
	t.Run("time lock 2", func(t *testing.T) {
		u := ledger.NewUTXODB(true)
		priv, pub := u.OriginKeys()
		t.Logf("orig priv key: %s", easyfl.Fmt(priv))
		t.Logf("orig pub key: %s", easyfl.Fmt(pub))
		t.Logf("origin address: %s", easyfl.Fmt(u.OriginAddress()))

		privKey0, _, addr0 := u.GenerateAddress(0)
		err := u.TokensFromFaucet(addr0, 10000)
		require.EqualValues(t, 1, u.NumUTXOs(u.OriginAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.OriginAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 1, u.NumUTXOs(addr0))

		priv1, _, addr1 := u.GenerateAddress(1)

		ts := uint32(time.Now().Unix()) + 5
		txBytes, err := ledger.MakeTransferTransaction(u, ledger.TransferTransactionParams{
			SenderKey:         privKey0,
			TargetAddress:     addr1,
			Amount:            200,
			Timestamp:         ts,
			AddTimeLockForSec: 1,
		})
		require.NoError(t, err)
		t.Logf("tx with timelock len: %d", len(txBytes))
		err = u.AddTransaction(txBytes, ledger.TraceOptionFailedConstraints)
		require.NoError(t, err)

		require.EqualValues(t, 200, u.Balance(addr1))

		err = u.DoTransfer(ledger.TransferTransactionParams{
			SenderKey:         privKey0,
			TargetAddress:     addr1,
			Amount:            2000,
			AddTimeLockForSec: 10,
			Timestamp:         ts + 1,
		})
		require.NoError(t, err)

		require.EqualValues(t, 2200, u.Balance(addr1))
		err = u.DoTransfer(ledger.TransferTransactionParams{
			SenderKey:     priv1,
			TargetAddress: addr0,
			Amount:        2000,
			Timestamp:     ts + 2,
		})
		easyfl.RequireErrorWith(t, err, "constraint 'timelock' failed")
		require.EqualValues(t, 2200, u.Balance(addr1))

		err = u.DoTransfer(ledger.TransferTransactionParams{
			SenderKey:     priv1,
			TargetAddress: addr0,
			Amount:        2000,
			Timestamp:     ts + 12,
		})
		require.NoError(t, err)
		require.EqualValues(t, 200, u.Balance(addr1))
	})
}
