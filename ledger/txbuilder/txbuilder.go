package txbuilder

import (
	"crypto"
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/library"
	"golang.org/x/crypto/blake2b"
)

type (
	TransactionBuilder struct {
		ConsumedOutputs []*Output
		Transaction     *transaction
	}

	transaction struct {
		InputIDs        []*ledger.OutputID
		Outputs         []*Output
		UnlockBlocks    []*UnlockParams
		Signature       []byte
		Timestamp       uint32
		InputCommitment [32]byte
	}

	UnlockParams struct {
		array *lazyslice.Array
	}
)

func NewTransactionBuilder() *TransactionBuilder {
	return &TransactionBuilder{
		ConsumedOutputs: make([]*Output, 0),
		Transaction: &transaction{
			InputIDs:        make([]*ledger.OutputID, 0),
			Outputs:         make([]*Output, 0),
			UnlockBlocks:    make([]*UnlockParams, 0),
			Timestamp:       0,
			InputCommitment: [32]byte{},
		},
	}
}

func (txb *TransactionBuilder) NumInputs() int {
	ret := len(txb.ConsumedOutputs)
	easyfl.Assert(ret == len(txb.Transaction.InputIDs), "ret==len(ctx.Transaction.InputIDs)")
	return ret
}

func (txb *TransactionBuilder) NumOutputs() int {
	return len(txb.Transaction.Outputs)
}

func (txb *TransactionBuilder) ConsumeOutput(out *Output, oid ledger.OutputID) (byte, error) {
	if txb.NumInputs() >= 256 {
		return 0, fmt.Errorf("too many consumed outputs")
	}
	txb.ConsumedOutputs = append(txb.ConsumedOutputs, out)
	txb.Transaction.InputIDs = append(txb.Transaction.InputIDs, &oid)
	txb.Transaction.UnlockBlocks = append(txb.Transaction.UnlockBlocks, NewUnlockBlock())

	return byte(len(txb.ConsumedOutputs) - 1), nil
}

func (txb *TransactionBuilder) PutUnlockParams(outputIndex, constraintIndex byte, unlockParamData []byte) {
	txb.Transaction.UnlockBlocks[outputIndex].array.PutAtIdxGrow(constraintIndex, unlockParamData)
}

// PutSignatureUnlock marker 0xff references signature of the transaction.
// It can be distinguished from any reference because it cannot be stringly less than any other reference
func (txb *TransactionBuilder) PutSignatureUnlock(outputIndex, constraintIndex byte) {
	txb.PutUnlockParams(outputIndex, constraintIndex, []byte{0xff})
}

// PutUnlockReference references some preceding output
func (txb *TransactionBuilder) PutUnlockReference(outputIndex, constraintIndex, referencedOutputIndex byte) error {
	if referencedOutputIndex >= outputIndex {
		return fmt.Errorf("referenced output index must be strongly less than the unlocked output index")
	}
	txb.PutUnlockParams(outputIndex, constraintIndex, []byte{referencedOutputIndex})
	return nil
}

func (txb *TransactionBuilder) ProduceOutput(out *Output) (byte, error) {
	if txb.NumOutputs() >= 256 {
		return 0, fmt.Errorf("too many produced outputs")
	}
	txb.Transaction.Outputs = append(txb.Transaction.Outputs, out)
	return byte(len(txb.Transaction.Outputs) - 1), nil
}

func (txb *TransactionBuilder) InputCommitment() [32]byte {
	arr := lazyslice.EmptyArray(256)
	for _, o := range txb.ConsumedOutputs {
		b := o.Bytes()
		arr.Push(b)
	}
	return blake2b.Sum256(arr.Bytes())
}

func (tx *transaction) ToArray() *lazyslice.Array {
	unlockParams := lazyslice.EmptyArray(256)
	inputIDs := lazyslice.EmptyArray(256)
	outputs := lazyslice.EmptyArray(256)

	for _, b := range tx.UnlockBlocks {
		unlockParams.Push(b.Bytes())
	}
	for _, oid := range tx.InputIDs {
		inputIDs.Push(oid[:])
	}
	for _, o := range tx.Outputs {
		outputs.Push(o.Bytes())
	}

	elems := make([]interface{}, library.TxTreeIndexMax)
	elems[library.TxUnlockParams] = unlockParams
	elems[library.TxInputIDs] = inputIDs
	elems[library.TxOutputs] = outputs
	elems[library.TxSignature] = tx.Signature
	var ts [4]byte
	binary.BigEndian.PutUint32(ts[:], tx.Timestamp)
	elems[library.TxTimestamp] = ts[:]
	elems[library.TxInputCommitment] = tx.InputCommitment[:]
	return lazyslice.MakeArray(elems...)
}

func (tx *transaction) Bytes() []byte {
	return tx.ToArray().Bytes()
}

func (tx *transaction) EssenceBytes() []byte {
	arr := tx.ToArray()
	return easyfl.Concat(
		arr.At(int(library.TxInputIDs)),
		arr.At(int(library.TxOutputs)),
		arr.At(int(library.TxInputCommitment)),
	)
}

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func (txb *TransactionBuilder) SignED25519(privKey ed25519.PrivateKey) {
	sig, err := privKey.Sign(rnd, txb.Transaction.EssenceBytes(), crypto.Hash(0))
	easyfl.AssertNoError(err)
	pubKey := privKey.Public().(ed25519.PublicKey)
	txb.Transaction.Signature = easyfl.Concat(sig, []byte(pubKey))
}

type TransferData struct {
	SenderPrivateKey ed25519.PrivateKey
	SenderPublicKey  ed25519.PublicKey
	SourceAccount    library.Accountable
	Outputs          []*OutputWithID
	ChainOutput      *OutputWithChainID
	Timestamp        uint32 // takes time.Now() if 0
	Lock             library.Lock
	Amount           uint64
	AdjustToMinimum  bool
	AddSender        bool
	AddConstraints   [][]byte
}

func NewTransferData(senderKey ed25519.PrivateKey, sourceAccount library.Accountable, ts uint32) *TransferData {
	sourcePubKey := senderKey.Public().(ed25519.PublicKey)
	if common.IsNil(sourceAccount) {
		sourceAccount = library.AddressED25519FromPublicKey(sourcePubKey)
	}
	return &TransferData{
		SenderPrivateKey: senderKey,
		SenderPublicKey:  sourcePubKey,
		SourceAccount:    sourceAccount,
		Timestamp:        ts,
		AddConstraints:   make([][]byte, 0),
	}
}

func (t *TransferData) WithTargetLock(lock library.Lock) *TransferData {
	t.Lock = lock
	return t
}

func (t *TransferData) WithAmount(amount uint64, adjustToMinimum ...bool) *TransferData {
	t.Amount = amount
	t.AdjustToMinimum = len(adjustToMinimum) > 0 && adjustToMinimum[0]
	return t
}

func (t *TransferData) WithConstraintBinary(constr []byte, idx ...byte) *TransferData {
	if len(idx) == 0 {
		t.AddConstraints = append(t.AddConstraints, constr)
	} else {
		easyfl.Assert(idx[0] == 0xff || idx[0] <= 2, "WithConstraintBinary: wrong constraint index")
		t.AddConstraints[idx[0]] = constr
	}
	return t
}

func (t *TransferData) WithConstraint(constr library.Constraint, idx ...byte) *TransferData {
	return t.WithConstraintBinary(constr.Bytes(), idx...)
}

func (t *TransferData) WithConstraintAtIndex(constr library.Constraint) *TransferData {
	return t.WithConstraintBinary(constr.Bytes())
}

func (t *TransferData) WithOutputs(outs []*OutputWithID) *TransferData {
	t.Outputs = outs
	return t
}

func (t *TransferData) WithChainOutput(out *OutputWithChainID) *TransferData {
	t.ChainOutput = out
	return t
}

func (t *TransferData) WithSender() *TransferData {
	t.AddSender = true
	return t
}

// AdjustedAmount adjust amount to minimum storage deposit requirements
func (t *TransferData) AdjustedAmount() uint64 {
	if !t.AdjustToMinimum {
		// not adjust. Will render wrong transaction if not enough tokens
		return t.Amount
	}
	ts := uint32(0)

	outTentative := NewOutput()
	outTentative.WithAmount(t.Amount)
	outTentative.WithTimestamp(ts)
	outTentative.WithLock(t.Lock)
	for _, c := range t.AddConstraints {
		_, err := outTentative.PushConstraint(c)
		easyfl.AssertNoError(err)
	}
	minimumDeposit := library.MinimumStorageDeposit(uint32(len(outTentative.Bytes())), 0)
	if t.Amount < minimumDeposit {
		return minimumDeposit
	}
	return t.Amount
}

func MakeTransferTransaction(par *TransferData) ([]byte, error) {
	var err error
	var ret []byte
	if par.ChainOutput == nil {
		ret, _, err = MakeSimpleTransferTransactionOutputs(par)
	} else {
		ret, _, err = MakeChainTransferTransactionOutputs(par)
	}
	return ret, err
}

func outputsToConsumeSimple(par *TransferData, amount uint64) (uint64, uint32, []*OutputWithID, error) {
	ts := uint32(time.Now().Unix())
	if par.Timestamp > 0 {
		ts = par.Timestamp
	}
	consumedOuts := par.Outputs[:0]
	availableTokens := uint64(0)
	numConsumedOutputs := 0

	for _, o := range par.Outputs {
		if numConsumedOutputs >= 256 {
			return 0, 0, nil, fmt.Errorf("exceeded max number of consumed outputs 256")
		}
		consumedOuts = append(consumedOuts, o)
		if o.Output.Timestamp() >= ts {
			ts = o.Output.Timestamp() + 1
		}
		numConsumedOutputs++
		availableTokens += o.Output.Amount()
		if availableTokens >= amount {
			break
		}
	}
	return availableTokens, ts, consumedOuts, nil
}

func MakeSimpleTransferTransactionOutputs(par *TransferData) ([]byte, []*ledger.OutputDataWithID, error) {
	if par.ChainOutput != nil {
		return nil, nil, fmt.Errorf("ChainOutput must be nil. Use MakeSimpleTransferTransactionOutputs instead")
	}
	amount := par.AdjustedAmount()
	availableTokens, ts, consumedOuts, err := outputsToConsumeSimple(par, amount)
	if err != nil {
		return nil, nil, err
	}
	if availableTokens < amount {
		if availableTokens < amount {
			return nil, nil, fmt.Errorf("not enough tokens in account %s: needed %d, got %d",
				par.SourceAccount.String(), par.Amount, availableTokens)
		}
	}

	txb := NewTransactionBuilder()
	for _, o := range consumedOuts {
		if _, err = txb.ConsumeOutput(o.Output, o.ID); err != nil {
			return nil, nil, err
		}
	}
	mainOutput := NewOutput().
		WithAmount(amount).
		WithTimestamp(ts).
		WithLock(par.Lock)

	if par.AddSender {
		senderAddr := library.AddressED25519FromPublicKey(par.SenderPublicKey)
		if _, err = mainOutput.PushConstraint(library.NewSenderAddressED25519(senderAddr).Bytes()); err != nil {
			return nil, nil, err
		}
	}

	for _, constr := range par.AddConstraints {
		if _, err = mainOutput.PushConstraint(constr); err != nil {
			return nil, nil, err
		}
	}
	var reminderOut *Output
	if availableTokens > amount {
		reminderOut = NewOutput().
			WithAmount(availableTokens - amount).
			WithTimestamp(ts).
			WithLock(par.SourceAccount.AsLock())
	}
	if reminderOut != nil {
		if _, err = txb.ProduceOutput(reminderOut); err != nil {
			return nil, nil, err
		}
	}
	if _, err = txb.ProduceOutput(mainOutput); err != nil {
		return nil, nil, err
	}

	for i := range consumedOuts {
		if i == 0 {
			txb.PutSignatureUnlock(0, library.ConstraintIndexLock)
		} else {
			// always referencing the 0 output
			err = txb.PutUnlockReference(byte(i), library.ConstraintIndexLock, 0)
			easyfl.AssertNoError(err)
		}
	}

	txb.Transaction.Timestamp = ts
	txb.Transaction.InputCommitment = txb.InputCommitment()
	txb.SignED25519(par.SenderPrivateKey)

	retOut := make([]*ledger.OutputDataWithID, 0)
	txBytes := txb.Transaction.Bytes()
	txid := ledger.TransactionID(blake2b.Sum256(txBytes))

	for i, o := range txb.Transaction.Outputs {
		retOut = append(retOut, &ledger.OutputDataWithID{
			ID:         ledger.NewOutputID(txid, byte(i)),
			OutputData: o.Bytes(),
		})
	}
	return txBytes, retOut, nil
}

func MakeChainTransferTransactionOutputs(par *TransferData) ([]byte, []*ledger.OutputDataWithID, error) {
	if par.ChainOutput == nil {
		return nil, nil, fmt.Errorf("ChainOutput must be provided")
	}
	amount := par.AdjustedAmount()
	// we are trying to consume non-chain outputs for the amount. Only if it is not enough, we are taking tokens from the chain
	availableTokens, ts, consumedOuts, err := outputsToConsumeSimple(par, amount)
	if err != nil {
		return nil, nil, err
	}
	// count the chain output in
	availableTokens += par.ChainOutput.Output.Amount()
	// some tokens must remain in the chain account
	if availableTokens <= amount {
		return nil, nil, fmt.Errorf("not enough tokens in account %s: needed %d, got %d",
			par.SourceAccount.String(), par.Amount, availableTokens)
	}

	txb := NewTransactionBuilder()

	if _, err = txb.ConsumeOutput(par.ChainOutput.Output, par.ChainOutput.ID); err != nil {
		return nil, nil, err
	}
	for _, o := range consumedOuts {
		if _, err = txb.ConsumeOutput(o.Output, o.ID); err != nil {
			return nil, nil, err
		}
	}
	chainConstr := library.NewChainConstraint(par.ChainOutput.ChainID, 0, par.ChainOutput.PredecessorConstraintIndex, 0)
	easyfl.Assert(availableTokens > amount, "availableTokens > amount")
	chainSuccessorOutput := par.ChainOutput.Output.Clone().
		WithAmount(availableTokens - amount).
		WithTimestamp(ts)
	chainSuccessorOutput.PutConstraint(chainConstr.Bytes(), par.ChainOutput.PredecessorConstraintIndex)
	if _, err = txb.ProduceOutput(chainSuccessorOutput); err != nil {
		return nil, nil, err
	}

	mainOutput := NewOutput().
		WithAmount(amount).
		WithTimestamp(ts).
		WithLock(par.Lock)

	if par.AddSender {
		senderAddr := library.AddressED25519FromPublicKey(par.SenderPublicKey)
		if _, err = mainOutput.PushConstraint(library.NewSenderAddressED25519(senderAddr).Bytes()); err != nil {
			return nil, nil, err
		}
	}
	for _, constr := range par.AddConstraints {
		if _, err = mainOutput.PushConstraint(constr); err != nil {
			return nil, nil, err
		}
	}
	if _, err = txb.ProduceOutput(mainOutput); err != nil {
		return nil, nil, err
	}
	// unlock chain input
	txb.PutSignatureUnlock(0, library.ConstraintIndexLock)
	txb.PutUnlockParams(0, par.ChainOutput.PredecessorConstraintIndex, []byte{0, par.ChainOutput.PredecessorConstraintIndex, 0})

	// always reference chain input
	for i := range consumedOuts {
		chainUnlockRef := library.NewChainLockUnlockParams(0, par.ChainOutput.PredecessorConstraintIndex)
		txb.PutUnlockParams(byte(i+1), library.ConstraintIndexLock, chainUnlockRef)
		easyfl.AssertNoError(err)
	}

	txb.Transaction.Timestamp = ts
	txb.Transaction.InputCommitment = txb.InputCommitment()
	txb.SignED25519(par.SenderPrivateKey)

	retOut := make([]*ledger.OutputDataWithID, 0)
	txBytes := txb.Transaction.Bytes()
	txid := ledger.TransactionID(blake2b.Sum256(txBytes))

	for i, o := range txb.Transaction.Outputs {
		retOut = append(retOut, &ledger.OutputDataWithID{
			ID:         ledger.NewOutputID(txid, byte(i)),
			OutputData: o.Bytes(),
		})
	}
	return txBytes, retOut, nil
}

//---------------------------------------------------------

func (u *UnlockParams) Bytes() []byte {
	return u.array.Bytes()
}

func NewUnlockBlock() *UnlockParams {
	return &UnlockParams{
		array: lazyslice.EmptyArray(256),
	}
}

func ForEachOutput(outs []*ledger.OutputDataWithID, fun func(o *Output, odata *ledger.OutputDataWithID) bool) error {
	for _, odata := range outs {
		o, err := OutputFromBytes(odata.OutputData)
		if err != nil {
			return err
		}
		if !fun(o, odata) {
			return nil
		}
	}
	return nil
}

func ParseChainConstraints(outs []*ledger.OutputDataWithID) ([]*OutputWithChainID, error) {
	ret := make([]*OutputWithChainID, 0)
	err := ForEachOutput(outs, func(o *Output, odata *ledger.OutputDataWithID) bool {
		ch, constraintIndex := o.ChainConstraint()
		if constraintIndex == 0xff {
			return true
		}
		d := &OutputWithChainID{
			OutputWithID: OutputWithID{
				ID:     odata.ID,
				Output: o,
			},
			PredecessorConstraintIndex: constraintIndex,
		}
		if ch.IsOrigin() {
			h := blake2b.Sum256(odata.ID[:])
			d.ChainID = h
		} else {
			d.ChainID = ch.ID
		}
		ret = append(ret, d)
		return true
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func GetChainAccount(chainID []byte, ind ledger.IndexerAccess, state ledger.StateAccess) (*OutputWithChainID, []*OutputWithID, error) {
	chainLock, err := library.ChainLockFromChainID(chainID)
	if err != nil {
		return nil, nil, err
	}
	chainOutData, err := ind.GetUTXOForChainID(chainID, state)
	if err != nil {
		return nil, nil, err
	}
	chainData, err := ParseChainConstraints([]*ledger.OutputDataWithID{chainOutData})
	if err != nil {
		return nil, nil, err
	}
	if len(chainData) != 1 {
		return nil, nil, fmt.Errorf("error while parsing chain output")
	}
	retData, err := ind.GetUTXOsLockedInAccount(chainLock, state)
	if err != nil {
		return nil, nil, err
	}
	ret, err := ParseAndSortOutputData(retData, nil)
	if err != nil {
		return nil, nil, err
	}
	return chainData[0], ret, nil
}

// InsertSimpleChainTransition inserts a simple chain transition. Takes output with chain constraint from parameters,
// Produces identical output, only modifies timestamp. Unlocks chain-input lock with signature reference
func (txb *TransactionBuilder) InsertSimpleChainTransition(inChainData *ledger.OutputDataWithChainID, ts uint32) error {
	chainIN, err := OutputFromBytes(inChainData.OutputData)
	if err != nil {
		return err
	}
	_, predecessorConstraintIndex := chainIN.ChainConstraint()
	if predecessorConstraintIndex == 0xff {
		return fmt.Errorf("can't find chain constrain in the output")
	}
	predecessorOutputIndex, err := txb.ConsumeOutput(chainIN, inChainData.ID)
	if err != nil {
		return err
	}
	chainOut := chainIN.Clone().WithTimestamp(ts)
	successor := library.NewChainConstraint(inChainData.ChainID, predecessorOutputIndex, predecessorConstraintIndex, 0)
	chainOut.PutConstraint(successor.Bytes(), predecessorConstraintIndex)
	successorOutputIndex, err := txb.ProduceOutput(chainOut)
	if err != nil {
		return err
	}
	txb.PutUnlockParams(predecessorOutputIndex, predecessorConstraintIndex, []byte{successorOutputIndex, predecessorConstraintIndex, 0})
	txb.PutSignatureUnlock(successorOutputIndex, library.ConstraintIndexLock)

	return nil
}
