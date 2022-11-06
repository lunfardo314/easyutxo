package ledger_test

import (
	"bytes"
	"crypto/ed25519"
	"math/rand"
	"testing"
	"time"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/library"
	"github.com/lunfardo314/easyutxo/ledger/state"
	"github.com/lunfardo314/easyutxo/ledger/txbuilder"
	"github.com/lunfardo314/easyutxo/ledger/utxodb"
	"github.com/stretchr/testify/require"
)

func TestOutput(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	pubKey, _, err := ed25519.GenerateKey(rnd)
	require.NoError(t, err)

	const msg = "message to be signed"

	t.Run("basic", func(t *testing.T) {
		out := txbuilder.OutputBasic(0, 0, library.AddressED25519Null())
		outBack, err := txbuilder.OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("empty output: %d bytes", len(out.Bytes()))
	})
	t.Run("address", func(t *testing.T) {
		out := txbuilder.OutputBasic(0, 0, library.AddressED25519FromPublicKey(pubKey))
		outBack, err := txbuilder.OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("output: %d bytes", len(out.Bytes()))

		_, err = library.AddressED25519FromBytes(outBack.Lock().Bytes())
		require.NoError(t, err)
		require.EqualValues(t, out.Lock(), outBack.Lock())
	})
	t.Run("tokens", func(t *testing.T) {
		out := txbuilder.OutputBasic(1337, uint32(time.Now().Unix()), library.AddressED25519Null())
		outBack, err := txbuilder.OutputFromBytes(out.Bytes())
		require.NoError(t, err)
		require.EqualValues(t, outBack.Bytes(), out.Bytes())
		t.Logf("output: %d bytes", len(out.Bytes()))

		tokensBack := outBack.Amount()
		require.EqualValues(t, 1337, tokensBack)
	})
}

func TestMainConstraints(t *testing.T) {
	t.Run("genesis", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)
		genesisBytes, found := u.StateAccess().GetUTXO(&ledger.OutputID{})
		require.True(t, found)
		out, err := txbuilder.OutputFromBytes(genesisBytes)
		require.NoError(t, err)
		require.EqualValues(t, u.Supply(), out.Amount())
		require.True(t, library.Equal(u.GenesisAddress(), out.Lock()))
		outsData, err := u.IndexerAccess().GetUTXOsForAccountID(u.GenesisAddress(), u.StateAccess())
		require.NoError(t, err)
		require.EqualValues(t, 1, len(outsData))
		require.EqualValues(t, ledger.OutputID{}, outsData[0].ID)
		require.EqualValues(t, genesisBytes, outsData[0].OutputData)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply(), u.Balance(u.GenesisAddress()))
	})
	t.Run("faucet", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)
		_, _, addr := u.GenerateAddress(1)
		err := u.TokensFromFaucet(addr, 10000)
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000, u.Balance(addr))
		require.EqualValues(t, 1, u.NumUTXOs(addr))
	})
	t.Run("simple transfer", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)
		privKey1, _, addr1 := u.GenerateAddress(1)
		err := u.TokensFromFaucet(addr1, 10000)
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000, u.Balance(addr1))
		require.EqualValues(t, 1, u.NumUTXOs(addr1))

		_, _, addrNext := u.GenerateAddress(2)
		in, err := u.MakeED25519TransferInputs(privKey1, 0)
		require.NoError(t, err)
		err = u.DoTransfer(in.WithTargetLock(addrNext).WithAmount(1000))
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000-1000, u.Balance(addr1))
		require.EqualValues(t, 1, u.NumUTXOs(addr1))
		require.EqualValues(t, 1000, u.Balance(addrNext))
		require.EqualValues(t, 1, u.NumUTXOs(addrNext))
	})
	t.Run("transfer wrong key", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)
		privKey1, _, addr1 := u.GenerateAddress(1)
		err := u.TokensFromFaucet(addr1, 10000)
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000, u.Balance(addr1))
		require.EqualValues(t, 1, u.NumUTXOs(addr1))

		_, _, addrNext := u.GenerateAddress(2)
		privKeyWrong, _, _ := u.GenerateAddress(3)
		in, err := u.MakeED25519TransferInputs(privKey1, 0)
		in.SenderPrivateKey = privKeyWrong
		require.NoError(t, err)
		err = u.DoTransfer(in.WithTargetLock(addrNext).WithAmount(1000))
		easyfl.RequireErrorWith(t, err, "failed")
	})
}

func TestTimelock(t *testing.T) {
	t.Run("time lock 1", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)
		privKey0, _, addr0 := u.GenerateAddress(0)
		err := u.TokensFromFaucet(addr0, 10000)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 1, u.NumUTXOs(addr0))

		priv1, _, addr1 := u.GenerateAddress(1)

		ts := uint32(time.Now().Unix()) + 5
		par, err := u.MakeED25519TransferInputs(privKey0, ts)
		require.NoError(t, err)
		par.WithAmount(200).
			WithTargetLock(addr1).
			WithConstraint(library.NewTimelock(ts + 1))
		txBytes, err := txbuilder.MakeTransferTransaction(par)

		require.NoError(t, err)
		t.Logf("tx with timelock len: %d", len(txBytes))
		err = u.AddTransaction(txBytes, state.TraceOptionFailedConstraints)
		require.NoError(t, err)

		require.EqualValues(t, 200, u.Balance(addr1))

		t.Logf("timelock: %x", ts+1)
		par, err = u.MakeED25519TransferInputs(privKey0, ts+1)
		require.NoError(t, err)
		par.WithAmount(2000).
			WithTargetLock(addr1).
			WithConstraint(library.NewTimelock(ts + 1 + 10))
		err = u.DoTransfer(par)
		require.NoError(t, err)

		require.EqualValues(t, 2200, u.Balance(addr1))

		par, err = u.MakeED25519TransferInputs(priv1, ts+2)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr0),
		)

		easyfl.RequireErrorWith(t, err, "failed")
		require.EqualValues(t, 2200, u.Balance(addr1))

		t.Logf("tx time: %x", ts+12)
		par, err = u.MakeED25519TransferInputs(priv1, ts+12)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr0),
		)
		require.NoError(t, err)
		require.EqualValues(t, 200, u.Balance(addr1))
	})
	t.Run("time lock 2", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)

		privKey0, _, addr0 := u.GenerateAddress(0)
		err := u.TokensFromFaucet(addr0, 10000)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 1, u.NumUTXOs(addr0))

		priv1, _, addr1 := u.GenerateAddress(1)

		ts := uint32(time.Now().Unix()) + 5
		par, err := u.MakeED25519TransferInputs(privKey0, ts)
		require.NoError(t, err)
		txBytes, err := txbuilder.MakeTransferTransaction(par.
			WithAmount(200).
			WithTargetLock(addr1).
			WithConstraint(library.NewTimelock(ts + 1)),
		)
		require.NoError(t, err)
		t.Logf("tx with timelock len: %d", len(txBytes))
		err = u.AddTransaction(txBytes, state.TraceOptionFailedConstraints)
		require.NoError(t, err)

		require.EqualValues(t, 200, u.Balance(addr1))

		par, err = u.MakeED25519TransferInputs(privKey0, ts+1)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr1).
			WithConstraint(library.NewTimelock(ts + 1 + 10)),
		)
		require.NoError(t, err)

		require.EqualValues(t, 2200, u.Balance(addr1))

		par, err = u.MakeED25519TransferInputs(priv1, ts+2)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr0),
		)
		easyfl.RequireErrorWith(t, err, "failed")
		require.EqualValues(t, 2200, u.Balance(addr1))

		par, err = u.MakeED25519TransferInputs(priv1, ts+12)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr0),
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
	require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
	require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
	require.EqualValues(t, 10000, u.Balance(addr0))
	require.EqualValues(t, 1, u.NumUTXOs(addr0))

	_, pubKey1, addr1 := u.GenerateAddress(1)
	require.EqualValues(t, 0, u.Balance(addr1))
	require.EqualValues(t, 0, u.NumUTXOs(addr1))

	ts := uint32(time.Now().Unix())

	par, err := u.MakeED25519TransferInputs(privKey0, ts)
	require.NoError(t, err)
	deadlineLock := library.NewDeadlineLock(
		ts+10,
		library.AddressED25519FromPublicKey(pubKey1),
		library.AddressED25519FromPublicKey(pubKey0),
	)
	t.Logf("deadline lock: %d bytes", len(deadlineLock.Bytes()))
	dis, err := easyfl.DecompileBinary(deadlineLock.Bytes())
	require.NoError(t, err)
	t.Logf("disassemble deadlock %s", dis)
	_, err = u.DoTransferTx(par.
		WithAmount(2000).
		WithTargetLock(deadlineLock),
	)
	require.NoError(t, err)

	require.EqualValues(t, 2, u.NumUTXOs(addr0))
	require.EqualValues(t, 10000, u.Balance(addr0))

	require.EqualValues(t, 1, u.NumUTXOs(addr0, ts+10))
	require.EqualValues(t, 2, u.NumUTXOs(addr0, ts+11))
	require.EqualValues(t, 8000, int(u.Balance(addr0, ts+10)))
	require.EqualValues(t, 10000, int(u.Balance(addr0, ts+11)))

	require.EqualValues(t, 1, u.NumUTXOs(addr1))
	require.EqualValues(t, 1, u.NumUTXOs(addr1, ts+10))
	require.EqualValues(t, 0, u.NumUTXOs(addr1, ts+11))
	require.EqualValues(t, 2000, int(u.Balance(addr1, ts+10)))
	require.EqualValues(t, 0, int(u.Balance(addr1, ts+11)))
}

func TestSenderAddressED25519(t *testing.T) {
	u := utxodb.NewUTXODB(true)
	privKey0, _, addr0 := u.GenerateAddress(0)
	err := u.TokensFromFaucet(addr0, 10000)
	require.NoError(t, err)
	require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
	require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
	require.EqualValues(t, 10000, u.Balance(addr0))
	require.EqualValues(t, 1, u.NumUTXOs(addr0))

	_, _, addr1 := u.GenerateAddress(1)
	require.EqualValues(t, 0, u.Balance(addr1))
	require.EqualValues(t, 0, u.NumUTXOs(addr1))

	par, err := u.MakeED25519TransferInputs(privKey0, uint32(time.Now().Unix()))
	err = u.DoTransfer(par.
		WithAmount(2000).
		WithTargetLock(addr1).
		WithSender(),
	)
	require.NoError(t, err)

	require.EqualValues(t, 1, u.NumUTXOs(addr1))
	require.EqualValues(t, 2000, u.Balance(addr1))

	outDatas, err := u.IndexerAccess().GetUTXOsForAccountID(addr1, u.StateAccess())
	require.NoError(t, err)
	outs, err := txbuilder.ParseAndSortOutputData(outDatas, nil)
	require.NoError(t, err)

	require.EqualValues(t, 1, len(outs))
	saddr, ok := outs[0].Output.SenderAddressED25519()
	require.True(t, ok)
	require.True(t, library.Equal(addr0, saddr))
}

func TestChain(t *testing.T) {
	var privKey0 ed25519.PrivateKey
	var u *utxodb.UTXODB
	var addr0 library.AddressED25519
	initTest := func() {
		u = utxodb.NewUTXODB(true)
		privKey0, _, addr0 = u.GenerateAddress(0)
		err := u.TokensFromFaucet(addr0, 10000)
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 1, u.NumUTXOs(addr0))
	}
	initTest2 := func() []*ledger.OutputDataWithChainID {
		initTest()
		par, err := u.MakeED25519TransferInputs(privKey0, uint32(time.Now().Unix()))
		outs, err := u.DoTransferOutputs(par.
			WithAmount(2000).
			WithTargetLock(addr0).
			WithConstraint(library.NewChainOrigin()),
		)
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 2, u.NumUTXOs(addr0))
		require.EqualValues(t, 2, len(outs))
		chains, err := txbuilder.ParseChainConstraints(outs)
		require.NoError(t, err)
		return chains
	}
	t.Run("compile", func(t *testing.T) {
		const source = "chain(originChainData)"
		_, _, _, err := easyfl.CompileExpression(source)
		require.NoError(t, err)
	})
	t.Run("create origin ok", func(t *testing.T) {
		initTest2()
	})
	t.Run("create origin ok 2", func(t *testing.T) {
		initTest()

		const source = "chain(originChainData)"
		_, _, code, err := easyfl.CompileExpression(source)
		require.NoError(t, err)

		par, err := u.MakeED25519TransferInputs(privKey0, uint32(time.Now().Unix()))
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr0).
			WithConstraintBinary(code),
		)
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 2, u.NumUTXOs(addr0))
	})
	t.Run("create origin twice in same output", func(t *testing.T) {
		initTest()

		const source = "chain(originChainData)"
		_, _, code, err := easyfl.CompileExpression(source)
		require.NoError(t, err)

		par, err := u.MakeED25519TransferInputs(privKey0, uint32(time.Now().Unix()))
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr0).
			WithConstraintBinary(code).
			WithConstraintBinary(code),
		)
		easyfl.RequireErrorWith(t, err, "duplicated constraints")
	})
	t.Run("create origin wrong 1", func(t *testing.T) {
		initTest()

		const source = "chain(0x0001)"
		_, _, code, err := easyfl.CompileExpression(source)
		require.NoError(t, err)

		par, err := u.MakeED25519TransferInputs(privKey0, uint32(time.Now().Unix()))
		par.WithAmount(2000).WithTargetLock(addr0)

		err = u.DoTransfer(par.WithConstraintBinary(code))
		require.Error(t, err)

		err = u.DoTransfer(par.WithConstraintBinary(bytes.Repeat([]byte{0}, 35)))
		require.Error(t, err)

		err = u.DoTransfer(par.WithConstraintBinary(nil))
		require.Error(t, err)
	})
	t.Run("create origin indexer", func(t *testing.T) {
		chains := initTest2()
		require.EqualValues(t, 1, len(chains))
		chs, err := u.IndexerAccess().GetUTXOForChainID(chains[0].ChainID[:], u.StateAccess())
		require.NoError(t, err)
		o, err := txbuilder.OutputFromBytes(chs.OutputData)
		require.NoError(t, err)
		ch, idx := o.ChainConstraint()
		require.True(t, idx != 0xff)
		require.True(t, ch.IsOrigin())
		t.Logf("chain created: %s", easyfl.Fmt(chains[0].ChainID[:]))
	})
	t.Run("create-destroy", func(t *testing.T) {
		chains := initTest2()
		require.EqualValues(t, 1, len(chains))
		chainID := chains[0].ChainID
		chs, err := u.IndexerAccess().GetUTXOForChainID(chainID[:], u.StateAccess())
		require.NoError(t, err)

		chainIN, err := txbuilder.OutputFromBytes(chs.OutputData)
		require.NoError(t, err)
		ch, predecessorConstraintIndex := chainIN.ChainConstraint()
		require.True(t, predecessorConstraintIndex != 0xff)
		require.True(t, ch.IsOrigin())
		t.Logf("chain created: %s", easyfl.Fmt(chains[0].ChainID[:]))

		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 2, u.NumUTXOs(addr0))

		ts := chainIN.Timestamp() + 1
		txb := txbuilder.NewTransactionBuilder()
		consumedIndex, err := txb.ConsumeOutput(chainIN, chains[0].ID)
		require.NoError(t, err)
		outNonChain := txbuilder.NewOutput().
			WithAmount(chainIN.Amount()).
			WithTimestamp(ts).
			WithLockConstraint(chainIN.Lock())
		_, err = txb.ProduceOutput(outNonChain)
		require.NoError(t, err)

		txb.Transaction.Timestamp = ts
		txb.Transaction.InputCommitment = txb.InputCommitment()

		txb.PutUnlockParams(consumedIndex, predecessorConstraintIndex, []byte{0xff, 0xff, 0xff})
		txb.PutSignatureUnlock(consumedIndex, library.ConstraintIndexLock)
		txb.SignED25519(privKey0)

		txbytes := txb.Transaction.Bytes()
		err = u.AddTransaction(txbytes, state.TraceOptionFailedConstraints)
		require.NoError(t, err)

		_, err = u.IndexerAccess().GetUTXOForChainID(chainID[:], u.StateAccess())
		easyfl.RequireErrorWith(t, err, "has not not been found")

		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 2, u.NumUTXOs(addr0))
	})
	t.Run("create-transit", func(t *testing.T) {

		// TODO double check

		chains := initTest2()
		require.EqualValues(t, 1, len(chains))
		chainID := chains[0].ChainID
		chs, err := u.IndexerAccess().GetUTXOForChainID(chainID[:], u.StateAccess())
		require.NoError(t, err)

		chainIN, err := txbuilder.OutputFromBytes(chs.OutputData)
		require.NoError(t, err)

		ts := chainIN.Timestamp() + 1
		txb := txbuilder.NewTransactionBuilder()
		err = txb.InsertChainTransition(chains[0], ts)
		require.NoError(t, err)

		txb.Transaction.Timestamp = ts
		txb.Transaction.InputCommitment = txb.InputCommitment()

		txb.SignED25519(privKey0)

		txbytes := txb.Transaction.Bytes()
		err = u.AddTransaction(txbytes, state.TraceOptionFailedConstraints)
		require.NoError(t, err)

		_, err = u.IndexerAccess().GetUTXOForChainID(chainID[:], u.StateAccess())
		require.NoError(t, err)

		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 2, u.NumUTXOs(addr0))
	})
}
