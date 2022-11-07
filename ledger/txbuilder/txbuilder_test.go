package txbuilder_test

import (
	"testing"
	"time"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger/txbuilder"
	"github.com/lunfardo314/easyutxo/ledger/utxodb"
	"github.com/stretchr/testify/require"
)

func TestBasics(t *testing.T) {
	t.Run("utxodb 1", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)
		priv, pub := u.GenesisKeys()
		t.Logf("orig priv key: %s", easyfl.Fmt(priv))
		t.Logf("orig pub key: %s", easyfl.Fmt(pub))
		t.Logf("origin address: %s", easyfl.Fmt(u.GenesisAddress()))

		_, _, addr := u.GenerateAddress(0)
		err := u.TokensFromFaucet(addr, 100)
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-100, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 100, u.Balance(addr))
		require.EqualValues(t, 1, u.NumUTXOs(addr))
	})
	t.Run("utxodb 2", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)
		priv, pub := u.GenesisKeys()
		t.Logf("orig priv key: %s", easyfl.Fmt(priv))
		t.Logf("orig pub key: %s", easyfl.Fmt(pub))
		t.Logf("origin address: %s", easyfl.Fmt(u.GenesisAddress()))

		privKey, _, addr := u.GenerateAddress(0)
		err := u.TokensFromFaucet(addr, 100)
		require.NoError(t, err)
		err = u.TokensFromFaucet(addr)
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-100-utxodb.TokensFromFaucetDefault, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 100+utxodb.TokensFromFaucetDefault, u.Balance(addr))
		require.EqualValues(t, 2, u.NumUTXOs(addr))

		err = u.TransferTokens(privKey, addr, u.Balance(addr))
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-100-utxodb.TokensFromFaucetDefault, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, 100+utxodb.TokensFromFaucetDefault, u.Balance(addr))
		require.EqualValues(t, 1, u.NumUTXOs(addr))
	})
	t.Run("utxodb 3 compress outputs", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)
		priv, pub := u.GenesisKeys()
		t.Logf("orig priv key: %s", easyfl.Fmt(priv))
		t.Logf("orig pub key: %s", easyfl.Fmt(pub))
		t.Logf("origin address: %s", easyfl.Fmt(u.GenesisAddress()))

		privKey, _, addr := u.GenerateAddress(0)
		const howMany = 256

		total := uint64(0)
		numOuts := 0
		for i := uint64(100); i <= howMany; i++ {
			err := u.TokensFromFaucet(addr, i)
			require.NoError(t, err)
			total += i
			numOuts++

			require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
			require.EqualValues(t, u.Supply()-total, u.Balance(u.GenesisAddress()))
			require.EqualValues(t, total, u.Balance(addr))
			require.EqualValues(t, numOuts, u.NumUTXOs(addr))
		}

		par, err := u.MakeED25519TransferInputs(privKey, uint32(time.Now().Unix()))
		require.NoError(t, err)
		txBytes, err := txbuilder.MakeTransferTransaction(par.
			WithAmount(u.Balance(addr)).
			WithTargetLock(addr),
		)
		require.NoError(t, err)
		t.Logf("tx size = %d bytes", len(txBytes))

		err = u.TransferTokens(privKey, addr, u.Balance(addr))
		require.NoError(t, err)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-total, u.Balance(u.GenesisAddress()))
		require.EqualValues(t, total, u.Balance(addr))
		require.EqualValues(t, 1, u.NumUTXOs(addr))
	})
	t.Run("utxodb too many inputs", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)
		priv, pub := u.GenesisKeys()
		t.Logf("orig priv key: %s", easyfl.Fmt(priv))
		t.Logf("orig pub key: %s", easyfl.Fmt(pub))
		t.Logf("origin address: %s", easyfl.Fmt(u.GenesisAddress()))

		privKey, _, addr := u.GenerateAddress(0)
		const howMany = 500

		total := uint64(0)
		numOuts := 0
		for i := uint64(100); i <= howMany; i++ {
			err := u.TokensFromFaucet(addr, i)
			require.NoError(t, err)
			total += i
			numOuts++

			require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
			require.EqualValues(t, u.Supply()-total, u.Balance(u.GenesisAddress()))
			require.EqualValues(t, total, u.Balance(addr))
			require.EqualValues(t, numOuts, u.NumUTXOs(addr))
		}
		err := u.TransferTokens(privKey, addr, u.Balance(addr))
		easyfl.RequireErrorWith(t, err, "exceeded max number of consumed outputs")
	})
	t.Run("utxodb fan out outputs", func(t *testing.T) {
		u := utxodb.NewUTXODB(true)
		priv, pub := u.GenesisKeys()
		t.Logf("orig priv key: %s", easyfl.Fmt(priv))
		t.Logf("orig pub key: %s", easyfl.Fmt(pub))
		t.Logf("origin address: %s", easyfl.Fmt(u.GenesisAddress()))

		privKey0, _, addr0 := u.GenerateAddress(0)
		const howMany = 100
		err := u.TokensFromFaucet(addr0, howMany*100)
		require.EqualValues(t, 1, u.NumUTXOs(u.GenesisAddress()))
		require.EqualValues(t, u.Supply()-howMany*100, u.Balance(u.GenesisAddress()))
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

		outs, err := u.IndexerAccess().GetUTXOsForAccountID(addr1, u.StateAccess())
		require.NoError(t, err)
		require.EqualValues(t, howMany, len(outs))
		//for _, o := range outs {
		//	_, ok := o.Output.Sender()
		//	require.False(t, ok)
		//}

		err = u.TransferTokens(privKey1, addr0, howMany*100)
		require.EqualValues(t, howMany*100, u.Balance(addr0))
		require.EqualValues(t, 1, u.NumUTXOs(addr0))
		require.EqualValues(t, 0, u.Balance(addr1))
		require.EqualValues(t, 0, u.NumUTXOs(addr1))

		outs, err = u.IndexerAccess().GetUTXOsForAccountID(addr0, u.StateAccess())
		require.NoError(t, err)
		require.EqualValues(t, 1, len(outs))

		//snd, ok := outs[0].Output.Sender()
		//require.True(t, ok)
		//require.EqualValues(t, addr1, snd)
	})
}
