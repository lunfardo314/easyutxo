package utxodb

import (
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraint"
	"github.com/lunfardo314/easyutxo/ledger/indexer"
	"github.com/lunfardo314/easyutxo/ledger/state"
	"github.com/lunfardo314/easyutxo/ledger/txbuilder"
	"golang.org/x/crypto/blake2b"
)

// UTXODB is a ledger.FinalState with faucet

type UTXODB struct {
	state             *state.FinalState
	indexer           *indexer.Indexer
	supply            uint64
	genesisPrivateKey ed25519.PrivateKey
	genesisPublicKey  ed25519.PublicKey
	genesisAddress    constraint.AddressED25519
	trace             bool
}

const (
	// for determinism
	originPrivateKey        = "8ec47313c15c3a4443c41619735109b56bc818f4a6b71d6a1f186ec96d15f28f14117899305d99fb4775de9223ce9886cfaa3195da1e40c5db47c61266f04dd2"
	deterministicSeed       = "1234567890987654321"
	supplyForTesting        = uint64(1_000_000_000_000)
	TokensFromFaucetDefault = uint64(1_000_000)
)

func NewUTXODB(trace ...bool) *UTXODB {
	originPrivateKeyBin, err := hex.DecodeString(originPrivateKey)
	if err != nil {
		panic(err)
	}
	originPubKey := ed25519.PrivateKey(originPrivateKeyBin).Public().(ed25519.PublicKey)
	if err != nil {
		panic(err)
	}
	originAddr := constraint.AddressED25519FromPublicKey(originPubKey)
	ret := &UTXODB{
		state:             state.NewInMemory(originAddr, supplyForTesting),
		indexer:           indexer.NewInMemory(originAddr),
		supply:            supplyForTesting,
		genesisPrivateKey: ed25519.PrivateKey(originPrivateKeyBin),
		genesisPublicKey:  originPubKey,
		genesisAddress:    originAddr,
		trace:             len(trace) > 0 && trace[0],
	}
	return ret
}

func (u *UTXODB) Supply() uint64 {
	return u.supply
}

func (u *UTXODB) StateAccess() ledger.StateAccess {
	return u.state
}

func (u *UTXODB) IndexerAccess() ledger.IndexerAccess {
	return u.indexer
}

func (u *UTXODB) GenesisKeys() (ed25519.PrivateKey, ed25519.PublicKey) {
	return u.genesisPrivateKey, u.genesisPublicKey
}

func (u *UTXODB) GenesisAddress() constraint.AddressED25519 {
	return u.genesisAddress
}

// AddTransaction validates transaction and updates ledger state and indexer
// Ledger state and indexer are on different transactions, so ledger state can
// succeed while indexer fails. In that case indexer can be updated from ledger state
func (u *UTXODB) AddTransaction(txBytes []byte, traceOption ...int) error {
	indexerUpdate, err := u.state.AddTransaction(txBytes, traceOption...)
	if err != nil {
		return err
	}
	if err = u.indexer.Update(indexerUpdate); err != nil {
		return fmt.Errorf("ledger state was updated but indexer update failed with '%v'", err)
	}
	return nil
}

func (u *UTXODB) TokensFromFaucet(addr constraint.AddressED25519, howMany ...uint64) error {
	amount := TokensFromFaucetDefault
	if len(howMany) > 0 && howMany[0] > 0 {
		amount = howMany[0]
	}
	outsData, err := u.indexer.GetUTXOsForAccountID(u.genesisAddress, u.state)
	if err != nil {
		return err
	}
	outs, err := txbuilder.ParseAndSortOutputData(outsData)
	if err != nil {
		return err
	}
	par := txbuilder.NewED25519TransferInputs(u.genesisPrivateKey).
		WithAmount(amount, true).
		WithTargetLock(addr).
		WithOutputs(outs)
	txBytes, err := txbuilder.MakeTransferTransaction(par)
	if err != nil {
		return fmt.Errorf("UTXODB faucet: %v", err)
	}

	trace := state.TraceOptionNone
	if u.trace {
		trace = state.TraceOptionFailedConstraints
	}
	return u.AddTransaction(txBytes, trace)
}

func (u *UTXODB) GenerateAddress(n uint16) (ed25519.PrivateKey, ed25519.PublicKey, constraint.AddressED25519) {
	var u16 [2]byte
	binary.BigEndian.PutUint16(u16[:], n)
	seed := blake2b.Sum256(common.Concat([]byte(deterministicSeed), u16[:]))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)
	addr := constraint.AddressED25519FromPublicKey(pub)
	return priv, pub, addr
}

func (u *UTXODB) MakeED25519TransferInputs(privKey ed25519.PrivateKey, desc ...bool) (*txbuilder.ED25519TransferInputs, error) {
	ret := txbuilder.NewED25519TransferInputs(privKey)
	outsData, err := u.indexer.GetUTXOsForAccountID(ret.SenderAddress, u.state)
	if err != nil {
		return nil, err
	}
	outs, err := txbuilder.ParseAndSortOutputData(outsData, desc...)
	if err != nil {
		return nil, err
	}
	ret.WithOutputs(outs)
	return ret, nil
}

func (u *UTXODB) TransferTokens(privKey ed25519.PrivateKey, targetLock constraint.Lock, amount uint64) error {
	par, err := u.MakeED25519TransferInputs(privKey)
	if err != nil {
		return err
	}
	par.WithAmount(amount).
		WithTargetLock(targetLock)
	//if len(addSender) > 0 && addSender[0] {
	//	par.WithConstraint(constraint.SenderConstraintBin(par.SenderAddress, 0))
	//}
	txBytes, err := txbuilder.MakeTransferTransaction(par)
	if err != nil {
		return err
	}
	trace := state.TraceOptionNone
	if u.trace {
		trace = state.TraceOptionFailedConstraints
	}
	return u.AddTransaction(txBytes, trace)
}

func (u *UTXODB) account(addr constraint.Accountable) (uint64, int) {
	outs, err := u.indexer.GetUTXOsForAccountID(addr, u.state)
	easyfl.AssertNoError(err)
	balance := uint64(0)
	for _, o := range outs {
		out, err := txbuilder.OutputFromBytes(o.OutputData)
		easyfl.AssertNoError(err)
		balance += out.Amount()
	}
	return balance, len(outs)
}

func (u *UTXODB) Balance(addr constraint.Accountable) uint64 {
	ret, _ := u.account(addr)
	return ret
}

func (u *UTXODB) NumUTXOs(addr constraint.Accountable) int {
	_, ret := u.account(addr)
	return ret
}

func (u *UTXODB) DoTransferTx(par *txbuilder.ED25519TransferInputs, traceOption ...int) ([]byte, error) {
	txBytes, err := txbuilder.MakeTransferTransaction(par)
	if err != nil {
		return nil, err
	}
	return txBytes, u.AddTransaction(txBytes, traceOption...)
}

func (u *UTXODB) DoTransfer(par *txbuilder.ED25519TransferInputs, traceOption ...int) error {
	_, err := u.DoTransferTx(par, traceOption...)
	return err
}

func (u *UTXODB) ValidationContextFromTransaction(txBytes []byte) (*state.ValidationContext, error) {
	return state.ValidationContextFromTransaction(txBytes, u.state)
}
