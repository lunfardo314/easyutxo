package ledger

import (
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/iotaledger/trie.go/common"
	"golang.org/x/crypto/blake2b"
)

// UTXODB is a ledger.State with faucet

type UTXODB struct {
	State
	originPrivateKey ed25519.PrivateKey
	originPublicKey  ed25519.PublicKey
	originAddress    Address
}

const (
	// for determinism
	originPrivateKey  = "8ec47313c15c3a4443c41619735109b56bc818f4a6b71d6a1f186ec96d15f28f14117899305d99fb4775de9223ce9886cfaa3195da1e40c5db47c61266f04dd2"
	deterministicSeed = "1234567890987654321"
	supplyForTesting  = uint64(1_000_000_000_000)
	tokensFromFaucet  = uint64(1_000_000)
)

func NewUTXODB() *UTXODB {
	originPrivKeyBin, err := hex.DecodeString(originPrivateKey)
	if err != nil {
		panic(err)
	}
	originPubKey := ed25519.PrivateKey(originPrivKeyBin).Public().(ed25519.PublicKey)
	if err != nil {
		panic(err)
	}
	ret := &UTXODB{
		State:            *NewLedgerStateInMemory(originPubKey, supplyForTesting),
		originPrivateKey: ed25519.PrivateKey(originPrivKeyBin),
		originPublicKey:  originPubKey,
		originAddress:    AddressFromED25519PubKey(originPubKey),
	}
	return ret
}

func (u *UTXODB) OriginKeys() (ed25519.PrivateKey, ed25519.PublicKey) {
	return u.originPrivateKey, u.originPublicKey
}

func (u *UTXODB) OriginAddress() Address {
	return u.originAddress
}

func (u *UTXODB) TokensFromFaucet(addr Address, howMany ...uint64) {
	amount := tokensFromFaucet
	if len(howMany) > 0 && howMany[0] > 0 {
		amount = howMany[0]
	}
	outs, err := u.GetUTXOsForAddress(u.OriginAddress())
	common.AssertNoError(err)
	common.Assert(len(outs) == 1, "len(outs)==1")
	origin := outs[0]
	common.Assert(origin.Output.Amount > amount, "UTXODB faucet is exhausted")

	ctx := NewTransactionContext()
	_, err = ctx.ConsumeOutput(origin.Output, origin.ID)
	common.AssertNoError(err)

	ts := uint32(time.Now().Unix())
	out := NewOutput()
	out.Timestamp = ts
	out.Amount = amount
	out.Address = Address{}

	reminder := NewOutput()
	reminder.Timestamp = ts
	reminder.Amount = origin.Output.Amount - amount
	reminder.Address = u.OriginAddress()

	_, err = ctx.ProduceOutput(out)
	common.AssertNoError(err)
	_, err = ctx.ProduceOutput(reminder)
	common.AssertNoError(err)

	err = u.AddTransaction(ctx.Transaction.Bytes())
	common.AssertNoError(err)
}

func (u *UTXODB) GenerateAddress(n uint16) (ed25519.PrivateKey, ed25519.PublicKey, Address) {
	var u16 [2]byte
	binary.BigEndian.PutUint16(u16[:], n)
	seed := blake2b.Sum256(common.Concat([]byte(deterministicSeed), u16[:]))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)
	addr := AddressFromED25519PubKey(pub)
	return priv, pub, addr
}
