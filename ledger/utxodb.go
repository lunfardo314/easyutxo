package ledger

import (
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"golang.org/x/crypto/blake2b"
)

// UTXODB is a ledger.State with faucet

type UTXODB struct {
	State
	originPrivateKey ed25519.PrivateKey
	originPublicKey  ed25519.PublicKey
	originAddress    Address
	trace            bool
}

const (
	// for determinism
	originPrivateKey  = "8ec47313c15c3a4443c41619735109b56bc818f4a6b71d6a1f186ec96d15f28f14117899305d99fb4775de9223ce9886cfaa3195da1e40c5db47c61266f04dd2"
	deterministicSeed = "1234567890987654321"
	supplyForTesting  = uint64(1_000_000_000_000)
	tokensFromFaucet  = uint64(1_000_000)
)

func NewUTXODB(trace ...bool) *UTXODB {
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
		trace:            len(trace) > 0 && trace[0],
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
	easyfl.AssertNoError(err)
	easyfl.Assert(len(outs) == 1, "len(outs)==1")
	origin := outs[0]
	easyfl.Assert(origin.Output.Amount > amount, "UTXODB faucet is exhausted")

	ts := uint32(time.Now().Unix())
	if origin.Output.Timestamp >= ts {
		ts = origin.Output.Timestamp + 1
	}
	ctx := NewTransactionContext()
	consumedOutputIdx, err := ctx.ConsumeOutput(origin.Output, origin.ID)
	easyfl.AssertNoError(err)

	out := NewOutput(amount, ts, addr)

	reminder := NewOutput(origin.Output.Amount-amount, ts, u.OriginAddress())

	_, err = ctx.ProduceOutput(out)
	easyfl.AssertNoError(err)
	_, err = ctx.ProduceOutput(reminder)
	easyfl.AssertNoError(err)

	ctx.Transaction.Timestamp = ts
	ctx.Transaction.InputCommitment = ctx.InputCommitment()

	// we must unlock the only consumed (genesis) output
	unlockData := UnlockDataBySignatureED25519(ctx.Transaction.EssenceBytes(), u.originPrivateKey)
	ctx.UnlockBlock(consumedOutputIdx).PutUnlockParams(unlockData, OutputBlockAddress)

	trace := TraceOptionNone
	if u.trace {
		trace = TraceOptionFailedConstraints
	}
	err = u.AddTransaction(ctx.Transaction.Bytes(), trace)
	easyfl.AssertNoError(err)
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
