package ledger_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
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
	"golang.org/x/crypto/blake2b"
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
		outsData, err := u.IndexerAccess().GetUTXOsLockedInAccount(u.GenesisAddress(), u.StateAccess())
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
		in, err := u.MakeTransferData(privKey1, nil, 0)
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
		in, err := u.MakeTransferData(privKey1, nil, 0)
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
		par, err := u.MakeTransferData(privKey0, nil, ts)
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
		par, err = u.MakeTransferData(privKey0, nil, ts+1)
		require.NoError(t, err)
		par.WithAmount(2000).
			WithTargetLock(addr1).
			WithConstraint(library.NewTimelock(ts + 1 + 10))
		err = u.DoTransfer(par)
		require.NoError(t, err)

		require.EqualValues(t, 2200, u.Balance(addr1))

		par, err = u.MakeTransferData(priv1, nil, ts+2)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr0),
		)

		easyfl.RequireErrorWith(t, err, "failed")
		require.EqualValues(t, 2200, u.Balance(addr1))

		t.Logf("tx time: %x", ts+12)
		par, err = u.MakeTransferData(priv1, nil, ts+12)
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
		par, err := u.MakeTransferData(privKey0, nil, ts)
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

		par, err = u.MakeTransferData(privKey0, nil, ts+1)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr1).
			WithConstraint(library.NewTimelock(ts + 1 + 10)),
		)
		require.NoError(t, err)

		require.EqualValues(t, 2200, u.Balance(addr1))

		par, err = u.MakeTransferData(priv1, nil, ts+2)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(2000).
			WithTargetLock(addr0),
		)
		easyfl.RequireErrorWith(t, err, "failed")
		require.EqualValues(t, 2200, u.Balance(addr1))

		par, err = u.MakeTransferData(priv1, nil, ts+12)
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

	par, err := u.MakeTransferData(privKey0, nil, ts)
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

	par, err := u.MakeTransferData(privKey0, nil, uint32(time.Now().Unix()))
	err = u.DoTransfer(par.
		WithAmount(2000).
		WithTargetLock(addr1).
		WithSender(),
	)
	require.NoError(t, err)

	require.EqualValues(t, 1, u.NumUTXOs(addr1))
	require.EqualValues(t, 2000, u.Balance(addr1))

	outDatas, err := u.IndexerAccess().GetUTXOsLockedInAccount(addr1, u.StateAccess())
	require.NoError(t, err)
	outs, err := txbuilder.ParseAndSortOutputData(outDatas, nil)
	require.NoError(t, err)

	require.EqualValues(t, 1, len(outs))
	saddr, ok := outs[0].Output.SenderAddressED25519()
	require.True(t, ok)
	require.True(t, library.Equal(addr0, saddr))
}

func TestChain1(t *testing.T) {
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
	initTest2 := func() []*txbuilder.OutputWithChainID {
		initTest()
		par, err := u.MakeTransferData(privKey0, nil, uint32(time.Now().Unix()))
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

		par, err := u.MakeTransferData(privKey0, nil, uint32(time.Now().Unix()))
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

		par, err := u.MakeTransferData(privKey0, nil, uint32(time.Now().Unix()))
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

		par, err := u.MakeTransferData(privKey0, nil, uint32(time.Now().Unix()))
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
			WithLock(chainIN.Lock())
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
}

func TestChain2(t *testing.T) {
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
	initTest2 := func() []*txbuilder.OutputWithChainID {
		initTest()
		par, err := u.MakeTransferData(privKey0, nil, uint32(time.Now().Unix()))
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
	runOption := func(option1, option2 int) error {
		chains := initTest2()
		require.EqualValues(t, 1, len(chains))
		theChainData := chains[0]
		chainID := theChainData.ChainID
		chs, err := u.IndexerAccess().GetUTXOForChainID(chainID[:], u.StateAccess())
		require.NoError(t, err)

		chainIN, err := txbuilder.OutputFromBytes(chs.OutputData)
		require.NoError(t, err)

		_, constraintIdx := chainIN.ChainConstraint()
		require.True(t, constraintIdx != 0xff)

		ts := chainIN.Timestamp() + 1
		txb := txbuilder.NewTransactionBuilder()
		predIdx, err := txb.ConsumeOutput(chainIN, chains[0].ID)
		require.NoError(t, err)

		var nextChainConstraint *library.ChainConstraint
		// options of making it wrong
		switch option1 {
		case 0:
			// good
			nextChainConstraint = library.NewChainConstraint(theChainData.ChainID, predIdx, constraintIdx, 0)
		case 1:
			nextChainConstraint = library.NewChainConstraint(theChainData.ChainID, 0xff, constraintIdx, 0)
		case 2:
			nextChainConstraint = library.NewChainConstraint(theChainData.ChainID, predIdx, 0xff, 0)
		case 3:
			nextChainConstraint = library.NewChainConstraint(theChainData.ChainID, 0xff, 0xff, 0)
		case 4:
			nextChainConstraint = library.NewChainConstraint(theChainData.ChainID, predIdx, constraintIdx, 1)
		case 5:
			nextChainConstraint = library.NewChainConstraint(theChainData.ChainID, 0xff, 0xff, 0xff)
		default:
			panic("wrong test option 1")
		}

		chainOut := chainIN.Clone().WithTimestamp(ts)
		chainOut.PutConstraint(nextChainConstraint.Bytes(), constraintIdx)
		succIdx, err := txb.ProduceOutput(chainOut)
		require.NoError(t, err)

		// options of wrong unlock params
		switch option2 {
		case 0:
			// good
			txb.PutUnlockParams(predIdx, constraintIdx, []byte{succIdx, constraintIdx, 0})
		case 1:
			txb.PutUnlockParams(predIdx, constraintIdx, []byte{0xff, constraintIdx, 0})
		case 2:
			txb.PutUnlockParams(predIdx, constraintIdx, []byte{succIdx, 0xff, 0})
		case 3:
			txb.PutUnlockParams(predIdx, constraintIdx, []byte{0xff, 0xff, 0})
		case 4:
			txb.PutUnlockParams(predIdx, constraintIdx, []byte{succIdx, constraintIdx, 1})
		default:
			panic("wrong test option 2")
		}
		txb.PutSignatureUnlock(0, library.ConstraintIndexLock)

		txb.Transaction.Timestamp = ts
		txb.Transaction.InputCommitment = txb.InputCommitment()

		txb.SignED25519(privKey0)

		txbytes := txb.Transaction.Bytes()
		err = u.AddTransaction(txbytes, state.TraceOptionFailedConstraints)
		if err != nil {
			return err
		}

		_, err = u.IndexerAccess().GetUTXOForChainID(chainID[:], u.StateAccess())
		require.NoError(t, err)

		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 10000, u.Balance(addr0))
		require.EqualValues(t, 2, u.NumUTXOs(addr0))
		return nil
	}
	t.Run("transit 0,0", func(t *testing.T) {
		err := runOption(0, 0)
		require.NoError(t, err)
	})
	t.Run("transit 1,0", func(t *testing.T) {
		err := runOption(1, 0)
		require.Error(t, err)
	})
	t.Run("transit 2,0", func(t *testing.T) {
		err := runOption(2, 0)
		require.Error(t, err)
	})
	t.Run("transit 3,0", func(t *testing.T) {
		err := runOption(3, 0)
		require.Error(t, err)
	})
	t.Run("transit 4,0", func(t *testing.T) {
		err := runOption(4, 0)
		require.Error(t, err)
	})
	t.Run("transit 5,0", func(t *testing.T) {
		err := runOption(5, 0)
		require.Error(t, err)
	})
	t.Run("transit 0,1", func(t *testing.T) {
		err := runOption(0, 1)
		require.Error(t, err)
	})
	t.Run("transit 0,2", func(t *testing.T) {
		err := runOption(0, 2)
		require.Error(t, err)
	})
	t.Run("transit 0,3", func(t *testing.T) {
		err := runOption(0, 3)
		require.Error(t, err)
	})
	t.Run("transit 0,4", func(t *testing.T) {
		err := runOption(0, 4)
		require.Error(t, err)
	})
	t.Run("transit 4,4", func(t *testing.T) {
		err := runOption(4, 4)
		require.NoError(t, err)
	})
}

func TestChain3(t *testing.T) {
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
	initTest2 := func() []*txbuilder.OutputWithChainID {
		initTest()
		par, err := u.MakeTransferData(privKey0, nil, uint32(time.Now().Unix()))
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
	chains := initTest2()
	require.EqualValues(t, 1, len(chains))
	theChainData := chains[0]
	chainID := theChainData.ChainID
	chs, err := u.IndexerAccess().GetUTXOForChainID(chainID[:], u.StateAccess())
	require.NoError(t, err)

	chainIN, err := txbuilder.OutputFromBytes(chs.OutputData)
	require.NoError(t, err)

	_, constraintIdx := chainIN.ChainConstraint()
	require.True(t, constraintIdx != 0xff)

	ts := chainIN.Timestamp() + 1
	txb := txbuilder.NewTransactionBuilder()
	predIdx, err := txb.ConsumeOutput(chainIN, chains[0].ID)
	require.NoError(t, err)

	var nextChainConstraint *library.ChainConstraint
	nextChainConstraint = library.NewChainConstraint(theChainData.ChainID, predIdx, constraintIdx, 0)

	chainOut := chainIN.Clone().WithTimestamp(ts)
	chainOut.PutConstraint(nextChainConstraint.Bytes(), constraintIdx)
	succIdx, err := txb.ProduceOutput(chainOut)
	require.NoError(t, err)

	txb.PutUnlockParams(predIdx, constraintIdx, []byte{succIdx, constraintIdx, 0})
	txb.PutSignatureUnlock(0, library.ConstraintIndexLock)

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
}

func TestChainLock(t *testing.T) {
	var privKey0, privKey1 ed25519.PrivateKey
	var addr0, addr1 library.AddressED25519
	var u *utxodb.UTXODB
	var chainID [32]byte
	var chainAddr library.ChainLock

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
	initTest2 := func() *txbuilder.OutputWithChainID {
		initTest()
		par, err := u.MakeTransferData(privKey0, nil, uint32(time.Now().Unix()))
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
		require.EqualValues(t, 1, len(chains))

		chainID = chains[0].ChainID
		chainAddr, err = library.ChainLockFromChainID(chainID[:])
		require.NoError(t, err)
		require.EqualValues(t, chainID[:], chainAddr.ChainID())

		onLocked, onChainOut, err := u.BalanceOnChain(chainID[:])
		require.NoError(t, err)
		require.EqualValues(t, 0, onLocked)
		require.EqualValues(t, 2000, onChainOut)

		_, err = u.IndexerAccess().GetUTXOForChainID(chainID[:], u.StateAccess())
		require.NoError(t, err)

		privKey1, _, addr1 = u.GenerateAddress(1)
		err = u.TokensFromFaucet(addr1, 20000)
		require.NoError(t, err)
		require.EqualValues(t, 20000, u.Balance(addr1))
		return chains[0]
	}
	sendFun := func(amount uint64, ts uint32) {
		par, err := u.MakeTransferData(privKey1, nil, ts)
		require.NoError(t, err)
		err = u.DoTransfer(par.
			WithAmount(amount).
			WithTargetLock(chainAddr),
		)
		require.NoError(t, err)
	}
	t.Run("send", func(t *testing.T) {
		initTest2()
		require.EqualValues(t, 20000, u.Balance(addr1))

		ts := uint32(time.Now().Unix()) + 5

		sendFun(1000, ts)
		sendFun(2000, ts+1)
		require.EqualValues(t, 20000-3000, int(u.Balance(addr1)))
		require.EqualValues(t, 3000, u.Balance(chainAddr))
		require.EqualValues(t, 2, u.NumUTXOs(chainAddr))

		onLocked, onChainOut, err := u.BalanceOnChain(chainID[:])
		require.NoError(t, err)
		require.EqualValues(t, 3000, onLocked)
		require.EqualValues(t, 2000, onChainOut)

		outs, err := u.IndexerAccess().GetUTXOsLockedInAccount(chainAddr, u.StateAccess())
		require.NoError(t, err)
		require.EqualValues(t, 2, len(outs))

		require.EqualValues(t, 10000, int(u.Balance(addr0)))
		par, err := u.MakeTransferData(privKey0, chainAddr, ts)
		par.WithAmount(500).WithTargetLock(addr0)
		require.NoError(t, err)
		txBytes, err := txbuilder.MakeTransferTransaction(par)
		require.NoError(t, err)

		v, err := u.ValidationContextFromTransaction(txBytes)
		require.NoError(t, err)
		t.Logf("%s", txbuilder.ValidationContextToString(v))

		require.EqualValues(t, 10000, int(u.Balance(addr0)))
		err = u.AddTransaction(txBytes, state.TraceOptionFailedConstraints)
		require.NoError(t, err)

		onLocked, onChainOut, err = u.BalanceOnChain(chainID[:])
		require.NoError(t, err)
		require.EqualValues(t, 2000, int(onLocked))
		require.EqualValues(t, 2500, int(onChainOut))
		require.EqualValues(t, 11000, int(u.Balance(addr0))) // also includes 500 on chain
	})

}

func TestLocalLibrary(t *testing.T) {
	const source = `
 func fun1 : concat($0,$1)
 func fun2 : fun1(fun1($0,$1), fun1($0,$1))
 func fun3 : fun2($0, $0)
`
	libBin, err := library.CompileLocalLibrary(source)
	require.NoError(t, err)
	t.Run("1", func(t *testing.T) {
		src := fmt.Sprintf("callLocalLibrary(0x%s, 2, 5)", hex.EncodeToString(libBin))
		t.Logf("src = '%s', len = %d", src, len(libBin))
		easyfl.MustEqual(src, "0x05050505")
	})
	t.Run("2", func(t *testing.T) {
		src := fmt.Sprintf("callLocalLibrary(0x%s, 0, 5, 6)", hex.EncodeToString(libBin))
		t.Logf("src = '%s', len = %d", src, len(libBin))
		easyfl.MustEqual(src, "0x0506")
	})
	t.Run("3", func(t *testing.T) {
		src := fmt.Sprintf("callLocalLibrary(0x%s, 1, 5, 6)", hex.EncodeToString(libBin))
		t.Logf("src = '%s', len = %d", src, len(libBin))
		easyfl.MustEqual(src, "0x05060506")
	})
	t.Run("4", func(t *testing.T) {
		src := fmt.Sprintf("callLocalLibrary(0x%s, 3)", hex.EncodeToString(libBin))
		t.Logf("src = '%s', len = %d", src, len(libBin))
		easyfl.MustError(src)
	})
}

func TestHashUnlock(t *testing.T) {
	const secretUnlockScript = "func fun1: and" // always returns true
	libBin, err := library.CompileLocalLibrary(secretUnlockScript)
	require.NoError(t, err)
	t.Logf("library size: %d", len(libBin))
	libHash := blake2b.Sum256(libBin)
	t.Logf("library hash: %s", easyfl.Fmt(libHash[:]))

	u := utxodb.NewUTXODB(true)
	privKey0, _, addr0 := u.GenerateAddress(0)
	err = u.TokensFromFaucet(addr0, 10000)
	require.NoError(t, err)
	require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
	require.EqualValues(t, u.Supply()-10000, u.Balance(u.GenesisAddress()))
	require.EqualValues(t, 10000, u.Balance(addr0))
	require.EqualValues(t, 1, u.NumUTXOs(addr0))

	constraintSource := fmt.Sprintf("or(isPathToProducedOutput(@),callLocalLibrary(selfHashUnlock(0x%s), 0))", hex.EncodeToString(libHash[:]))
	_, _, constraintBin, err := easyfl.CompileExpression(constraintSource)
	require.NoError(t, err)
	t.Logf("constraint source: %s", constraintSource)
	t.Logf("constraint size: %d", len(constraintBin))

	par, err := u.MakeTransferData(privKey0, nil, 0)
	require.NoError(t, err)
	constr := library.NewGeneralScript(constraintBin)
	t.Logf("constraint: %s", constr)
	par.WithAmount(1000).
		WithTargetLock(addr0).
		WithConstraint(constr)
	txbytes, err := txbuilder.MakeTransferTransaction(par)
	require.NoError(t, err)

	ctx, err := state.ValidationContextFromTransaction(txbytes, u.StateAccess())
	require.NoError(t, err)

	t.Logf("%s", txbuilder.ValidationContextToString(ctx))
	outsData, err := u.DoTransferOutputs(par)
	require.NoError(t, err)

	outs, err := txbuilder.ParseAndSortOutputData(outsData, func(o *txbuilder.Output) bool {
		return o.Amount() == 1000
	})
	require.NoError(t, err)

	// produce transaction without providing hash unlocking library for the output with script
	par = txbuilder.NewTransferData(privKey0, addr0, 0)
	par.WithOutputs(outs).
		WithAmount(1000).
		WithTargetLock(addr0)

	txbytes, err = txbuilder.MakeTransferTransaction(par)
	require.NoError(t, err)

	ctx, err = state.ValidationContextFromTransaction(txbytes, u.StateAccess())
	require.NoError(t, err)

	t.Logf("---- transaction without hash unlock: FAILING %s", txbuilder.ValidationContextToString(ctx))
	err = u.DoTransfer(par)
	require.Error(t, err)

	// now adding unlock data the unlocking library/script
	par.WithUnlockData(0, 3, libBin)

	txbytes, err = txbuilder.MakeTransferTransaction(par)
	require.NoError(t, err)

	ctx, err = state.ValidationContextFromTransaction(txbytes, u.StateAccess())
	require.NoError(t, err)

	t.Logf("---- transaction with hash unlock, the library/script: SUCCESS %s", txbuilder.ValidationContextToString(ctx))
	t.Logf("%s", txbuilder.ValidationContextToString(ctx))
	err = u.DoTransfer(par)
	require.NoError(t, err)
}
