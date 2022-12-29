package utxodb

import (
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraints"
	"github.com/lunfardo314/easyutxo/ledger/indexer"
	"github.com/lunfardo314/easyutxo/ledger/state"
	"github.com/lunfardo314/easyutxo/ledger/txbuilder"
	"github.com/lunfardo314/unitrie/common"
	"golang.org/x/crypto/blake2b"
)

// UTXODB is a centralized ledger.Updatable with indexer and genesis faucet
type UTXODB struct {
	state             *state.Updatable
	indexer           *indexer.Indexer
	stateStore        ledger.StateStore
	indexerStore      ledger.IndexerStore
	root              common.VCommitment
	supply            uint64
	genesisPrivateKey ed25519.PrivateKey
	genesisPublicKey  ed25519.PublicKey
	genesisAddress    constraints.AddressED25519
	trace             bool
}

const (
	// for determinism
	originPrivateKey        = "8ec47313c15c3a4443c41619735109b56bc818f4a6b71d6a1f186ec96d15f28f14117899305d99fb4775de9223ce9886cfaa3195da1e40c5db47c61266f04dd2"
	deterministicSeed       = "1234567890987654321"
	supplyForTesting        = uint64(1_000_000_000_000)
	TokensFromFaucetDefault = uint64(1_000_000)
	utxodbIdentity          = "utxodb"
)

func NewUTXODB(trace ...bool) *UTXODB {
	genesisPrivateKeyBin, err := hex.DecodeString(originPrivateKey)
	if err != nil {
		panic(err)
	}
	genesisPubKey := ed25519.PrivateKey(genesisPrivateKeyBin).Public().(ed25519.PublicKey)
	if err != nil {
		panic(err)
	}
	genesisAddr := constraints.AddressED25519FromPublicKey(genesisPubKey)

	stateStore := common.NewInMemoryKVStore()
	indexerStore := common.NewInMemoryKVStore()
	ts := uint32(time.Now().Unix())
	root := state.InitLedgerState(stateStore, []byte(utxodbIdentity), supplyForTesting, genesisAddr, ts)

	indexer.InitIndexer(indexerStore, genesisAddr)

	ret := &UTXODB{
		stateStore:        stateStore,
		indexerStore:      indexerStore,
		root:              root,
		supply:            supplyForTesting,
		genesisPrivateKey: ed25519.PrivateKey(genesisPrivateKeyBin),
		genesisPublicKey:  genesisPubKey,
		genesisAddress:    genesisAddr,
		trace:             len(trace) > 0 && trace[0],
	}
	return ret
}

func genesisMilestoneOutput(g *ledger.GenesisDataStruct, genesisStateCommitment common.VCommitment) []byte {
	amount := constraints.NewAmount(g.MilestoneDeposit)
	timestamp := constraints.NewTimestamp(g.TimestampMilestone)
	chainConstraint := constraints.NewChainConstraint(g.MilestoneChainID, 0, 0, 0)
	stateCommitmentConstraint, err := constraints.NewGeneralScriptFromSource(fmt.Sprintf("id(0x%s)", genesisStateCommitment.String()))
	common.AssertNoError(err)

	ret := lazyslice.EmptyArray()
	ret.Push(amount.Bytes())
	ret.Push(timestamp.Bytes())
	ret.Push(g.MilestoneController.Bytes())
	ret.Push(chainConstraint.Bytes())
	ret.Push(stateCommitmentConstraint.Bytes())
	return ret.Bytes()
}

func (u *UTXODB) GenesisData() *ledger.GenesisDataStruct {
	return u.genesisData
}

func (u *UTXODB) Supply() uint64 {
	return u.supply
}

func (u *UTXODB) FinalStateRoot() common.VCommitment {
	indexer := indexer.New(u.indexerStore)
	state := state.N
	indexer.GetUTXOForChainID(u.genesisData.MilestoneChainID[:])
}

func (u *UTXODB) StateAccess() ledger.StateReadAccess {
	return u.stateStore.Readable()
}

func (u *UTXODB) IndexerAccess() ledger.IndexerAccess {
	return u.indexerStore
}

func (u *UTXODB) GenesisKeys() (ed25519.PrivateKey, ed25519.PublicKey) {
	return u.genesisPrivateKey, u.genesisPublicKey
}

func (u *UTXODB) GenesisAddress() constraints.AddressED25519 {
	return u.genesisAddress
}

// AddTransaction validates transaction and updates ledger state and indexer
// Ledger state and indexer are on different DB transactions, so ledger state can
// succeed while indexer fails. In that case indexer can be updated from ledger state
func (u *UTXODB) AddTransaction(txBytes []byte, traceOption ...int) error {
	indexerUpdate, err := u.stateStore.Update(txBytes, traceOption...)
	if err != nil {
		return err
	}
	if err = u.indexerStore.Update(indexerUpdate); err != nil {
		return fmt.Errorf("ledger state has been updated but indexer update failed with '%v'", err)
	}
	return nil
}

func (u *UTXODB) TokensFromFaucet(addr constraints.AddressED25519, howMany ...uint64) error {
	amount := TokensFromFaucetDefault
	if len(howMany) > 0 && howMany[0] > 0 {
		amount = howMany[0]
	}
	outsData, err := u.indexerStore.GetUTXOsLockedInAccount(u.genesisAddress, u.stateStore.Readable())
	if err != nil {
		return err
	}
	outs, err := txbuilder.ParseAndSortOutputData(outsData, nil)
	if err != nil {
		return err
	}
	par := txbuilder.NewTransferData(u.genesisPrivateKey, nil, uint32(time.Now().Unix())).
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

func (u *UTXODB) GenerateAddress(n uint16) (ed25519.PrivateKey, ed25519.PublicKey, constraints.AddressED25519) {
	var u16 [2]byte
	binary.BigEndian.PutUint16(u16[:], n)
	seed := blake2b.Sum256(common.Concat([]byte(deterministicSeed), u16[:]))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)
	addr := constraints.AddressED25519FromPublicKey(pub)
	return priv, pub, addr
}

func (u *UTXODB) MakeTransferData(privKey ed25519.PrivateKey, sourceAccount constraints.Accountable, ts uint32, desc ...bool) (*txbuilder.TransferData, error) {
	if ts == 0 {
		ts = uint32(time.Now().Unix())
	}
	ret := txbuilder.NewTransferData(privKey, sourceAccount, ts)

	switch addr := ret.SourceAccount.(type) {
	case constraints.AddressED25519:
		if err := u.makeTransferInputsED25519(ret, desc...); err != nil {
			return nil, err
		}
		return ret, nil
	case constraints.ChainLock:
		if err := u.makeTransferDataChainLock(ret, addr, desc...); err != nil {
			return nil, err
		}
	default:
		panic(fmt.Sprintf("wrong source account type %T", sourceAccount))
	}
	return ret, nil
}

func (u *UTXODB) makeTransferInputsED25519(par *txbuilder.TransferData, desc ...bool) error {
	outsData, err := u.indexerStore.GetUTXOsLockedInAccount(par.SourceAccount, u.stateStore.Readable())
	if err != nil {
		return err
	}
	outs, err := txbuilder.ParseAndSortOutputData(outsData, func(o *txbuilder.Output) bool {
		return o.Lock().UnlockableWith(par.SourceAccount.AccountID(), par.Timestamp)
	}, desc...)
	if err != nil {
		return err
	}
	par.WithOutputs(outs)
	return nil
}

func (u *UTXODB) makeTransferDataChainLock(par *txbuilder.TransferData, chainLock constraints.ChainLock, desc ...bool) error {
	outChain, outs, err := txbuilder.GetChainAccount(chainLock.ChainID(), u.IndexerAccess(), u.StateAccess(), desc...)
	if err != nil {
		return err
	}
	par.WithOutputs(outs).
		WithChainOutput(outChain)
	return nil
}

func (u *UTXODB) TransferTokens(privKey ed25519.PrivateKey, targetLock constraints.Lock, amount uint64) error {
	par, err := u.MakeTransferData(privKey, nil, 0)
	if err != nil {
		return err
	}
	par.WithAmount(amount).
		WithTargetLock(targetLock)
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

func (u *UTXODB) account(addr constraints.Accountable, ts ...uint32) (uint64, int) {
	outs, err := u.indexerStore.GetUTXOsLockedInAccount(addr, u.stateStore.Readable())
	easyfl.AssertNoError(err)
	balance := uint64(0)
	var filter func(o *txbuilder.Output) bool
	if len(ts) > 0 {
		filter = func(o *txbuilder.Output) bool {
			return o.Lock().UnlockableWith(addr.AccountID(), ts[0])
		}
	}
	outs1, err := txbuilder.ParseAndSortOutputData(outs, filter)
	easyfl.AssertNoError(err)

	for _, o := range outs1 {
		balance += o.Output.Amount()
	}
	return balance, len(outs1)
}

// Balance returns balance of address unlockable at timestamp ts, if provided. Otherwise, all outputs taken
// For chains, this does not include te chain-output itself
func (u *UTXODB) Balance(addr constraints.Accountable, ts ...uint32) uint64 {
	ret, _ := u.account(addr, ts...)
	return ret
}

// BalanceOnChain returns balance locked in chain and separately balance on chain output
func (u *UTXODB) BalanceOnChain(chainID []byte) (uint64, uint64, error) {
	outChain, outs, err := txbuilder.GetChainAccount(chainID, u.IndexerAccess(), u.StateAccess())
	if err != nil {
		return 0, 0, err
	}
	amount := uint64(0)
	for _, odata := range outs {
		amount += odata.Output.Amount()
	}
	return amount, outChain.Output.Amount(), nil
}

// NumUTXOs returns number of outputs of address unlockable at timestamp ts, if provided. Otherwise, all outputs taken
func (u *UTXODB) NumUTXOs(addr constraints.Accountable, ts ...uint32) int {
	_, ret := u.account(addr, ts...)
	return ret
}

func (u *UTXODB) DoTransferTx(par *txbuilder.TransferData) ([]byte, error) {
	trace := state.TraceOptionNone
	if u.trace {
		trace = state.TraceOptionFailedConstraints
	}
	txBytes, err := txbuilder.MakeTransferTransaction(par)
	if err != nil {
		return nil, err
	}
	return txBytes, u.AddTransaction(txBytes, trace)
}

func (u *UTXODB) DoTransferOutputs(par *txbuilder.TransferData) ([]*ledger.OutputDataWithID, error) {
	trace := state.TraceOptionNone
	if u.trace {
		trace = state.TraceOptionFailedConstraints
	}
	txBytes, retOuts, err := txbuilder.MakeSimpleTransferTransactionOutputs(par)
	if err != nil {
		return nil, err
	}
	if err = u.AddTransaction(txBytes, trace); err != nil {
		return nil, err
	}
	return retOuts, nil
}

func (u *UTXODB) DoTransfer(par *txbuilder.TransferData) error {
	_, err := u.DoTransferTx(par)
	return err
}

func (u *UTXODB) ValidationContextFromTransaction(txBytes []byte) (*state.TransactionContext, error) {
	return state.TransactionContextFromTransferableBytes(txBytes, u.stateStore.Readable())
}

func (u *UTXODB) TxToString(txbytes []byte) string {
	ctx, err := u.ValidationContextFromTransaction(txbytes)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return txbuilder.ValidationContextToString(ctx)
}
