package ledger

import (
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"golang.org/x/crypto/blake2b"
)

// UTXODB is a ledger.State with faucet

type UTXODB struct {
	State
	supply           uint64
	originPrivateKey ed25519.PrivateKey
	originPublicKey  ed25519.PublicKey
	originAddress    Lock
	trace            bool
}

const (
	// for determinism
	originPrivateKey        = "8ec47313c15c3a4443c41619735109b56bc818f4a6b71d6a1f186ec96d15f28f14117899305d99fb4775de9223ce9886cfaa3195da1e40c5db47c61266f04dd2"
	deterministicSeed       = "1234567890987654321"
	supplyForTesting        = uint64(1_000_000_000_000)
	TokensFromFaucetDefault = uint64(1_000_000)
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
		supply:           supplyForTesting,
		originPrivateKey: ed25519.PrivateKey(originPrivKeyBin),
		originPublicKey:  originPubKey,
		originAddress:    LockFromED25519PubKey(originPubKey),
		trace:            len(trace) > 0 && trace[0],
	}
	return ret
}

func (u *UTXODB) Supply() uint64 {
	return u.supply
}

func (u *UTXODB) OriginKeys() (ed25519.PrivateKey, ed25519.PublicKey) {
	return u.originPrivateKey, u.originPublicKey
}

func (u *UTXODB) OriginAddress() Lock {
	return u.originAddress
}

func (u *UTXODB) TokensFromFaucet(addr Lock, howMany ...uint64) error {
	amount := TokensFromFaucetDefault
	if len(howMany) > 0 && howMany[0] > 0 {
		amount = howMany[0]
	}
	txBytes, err := MakeTransferTransaction(u, TransferTransactionParams{
		SenderKey:     u.originPrivateKey,
		TargetAddress: addr,
		Amount:        amount,
	})
	if err != nil {
		return fmt.Errorf("UTXODB faucet: %v", err)
	}

	trace := TraceOptionNone
	if u.trace {
		trace = TraceOptionFailedConstraints
	}
	return u.AddTransaction(txBytes, trace)
}

func (u *UTXODB) GenerateAddress(n uint16) (ed25519.PrivateKey, ed25519.PublicKey, Lock) {
	var u16 [2]byte
	binary.BigEndian.PutUint16(u16[:], n)
	seed := blake2b.Sum256(common.Concat([]byte(deterministicSeed), u16[:]))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)
	addr := LockFromED25519PubKey(pub)
	return priv, pub, addr
}

func (u *UTXODB) TransferTokens(privKey ed25519.PrivateKey, targetAddress Lock, amount uint64, addSender ...bool) error {
	txBytes, err := MakeTransferTransaction(u, TransferTransactionParams{
		SenderKey:     privKey,
		TargetAddress: targetAddress,
		Amount:        amount,
		AddSender:     len(addSender) > 0 && addSender[0],
	})
	if err != nil {
		return err
	}
	trace := TraceOptionNone
	if u.trace {
		trace = TraceOptionFailedConstraints
	}
	return u.AddTransaction(txBytes, trace)
}

func (u *UTXODB) account(addr Lock) (uint64, int) {
	outs, err := u.GetUTXOsForAddress(addr)
	easyfl.AssertNoError(err)
	balance := uint64(0)
	for _, o := range outs {
		balance += o.Output.Amount
	}
	return balance, len(outs)
}

func (u *UTXODB) Balance(addr Lock) uint64 {
	ret, _ := u.account(addr)
	return ret
}

func (u *UTXODB) NumUTXOs(addr Lock) int {
	_, ret := u.account(addr)
	return ret
}
