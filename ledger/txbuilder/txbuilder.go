package txbuilder

import (
	"crypto"
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"

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

// InsertChainTransition inserts a simple chain transition. Takes output with chain constraint from parameters,
// Produces identical output, only modifies timestamp. Unlocks chain-input lock with signature reference
func (txb *TransactionBuilder) InsertChainTransition(inChainData *ledger.OutputDataWithChainID, ts uint32) error {
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
	successor := library.NewChainConstraint(inChainData.ChainID, 0, predecessorOutputIndex, predecessorConstraintIndex)
	chainOut.PutConstraint(successor.Bytes(), predecessorConstraintIndex)
	successorOutputIndex, err := txb.ProduceOutput(chainOut)
	if err != nil {
		return err
	}
	txb.PutUnlockParams(predecessorOutputIndex, predecessorConstraintIndex, []byte{successorOutputIndex, predecessorConstraintIndex, 0})
	txb.PutSignatureUnlock(successorOutputIndex, library.ConstraintIndexLock)

	return nil
}

type ED25519TransferInputs struct {
	SenderPrivateKey ed25519.PrivateKey
	SenderPublicKey  ed25519.PublicKey
	SenderAddress    library.AddressED25519
	Outputs          []*OutputWithID
	Timestamp        uint32 // takes time.Now() if 0
	Lock             library.Lock
	Amount           uint64
	AdjustToMinimum  bool
	AddSender        bool
	AddConstraints   [][]byte
}

func NewED25519TransferInputs(senderKey ed25519.PrivateKey, ts uint32) *ED25519TransferInputs {
	sourcePubKey := senderKey.Public().(ed25519.PublicKey)
	sourceAddr := library.AddressED25519FromPublicKey(sourcePubKey)
	return &ED25519TransferInputs{
		SenderPrivateKey: senderKey,
		SenderPublicKey:  sourcePubKey,
		SenderAddress:    sourceAddr,
		Timestamp:        ts,
		AddConstraints:   make([][]byte, 0),
	}
}

func (t *ED25519TransferInputs) WithTargetLock(lock library.Lock) *ED25519TransferInputs {
	t.Lock = lock
	return t
}

func (t *ED25519TransferInputs) WithAmount(amount uint64, adjustToMinimum ...bool) *ED25519TransferInputs {
	t.Amount = amount
	t.AdjustToMinimum = len(adjustToMinimum) > 0 && adjustToMinimum[0]
	return t
}

func (t *ED25519TransferInputs) WithConstraintBinary(constr []byte, idx ...byte) *ED25519TransferInputs {
	if len(idx) == 0 {
		t.AddConstraints = append(t.AddConstraints, constr)
	} else {
		easyfl.Assert(idx[0] == 0xff || idx[0] <= 2, "WithConstraintBinary: wrong constraint index")
		t.AddConstraints[idx[0]] = constr
	}
	return t
}

func (t *ED25519TransferInputs) WithConstraint(constr library.Constraint, idx ...byte) *ED25519TransferInputs {
	return t.WithConstraintBinary(constr.Bytes(), idx...)
}

func (t *ED25519TransferInputs) WithConstraintAtIndex(constr library.Constraint) *ED25519TransferInputs {
	return t.WithConstraintBinary(constr.Bytes())
}

func (t *ED25519TransferInputs) WithOutputs(outs []*OutputWithID) *ED25519TransferInputs {
	t.Outputs = outs
	return t
}

func (t *ED25519TransferInputs) WithSender() *ED25519TransferInputs {
	t.AddSender = true
	return t
}

// AdjustedAmount adjust amount to minimum storage deposit requirements
func (t *ED25519TransferInputs) AdjustedAmount() uint64 {
	if !t.AdjustToMinimum {
		// not adjust. Will render wrong transaction if not enough tokens
		return t.Amount
	}
	ts := uint32(0)

	outTentative := NewOutput()
	outTentative.WithAmount(t.Amount)
	outTentative.WithTimestamp(ts)
	outTentative.WithLockConstraint(t.Lock)
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

func MakeTransferTransaction(par *ED25519TransferInputs) ([]byte, error) {
	ret, _, err := MakeTransferTransactionOutputs(par)
	return ret, err
}

func MakeTransferTransactionOutputs(par *ED25519TransferInputs) ([]byte, []*ledger.OutputDataWithID, error) {
	ts := uint32(time.Now().Unix())
	if par.Timestamp > 0 {
		ts = par.Timestamp
	}
	amount := par.AdjustedAmount()
	consumedOuts := par.Outputs[:0]
	availableTokens := uint64(0)
	numConsumedOutputs := 0

	for _, o := range par.Outputs {
		if numConsumedOutputs >= 256 {
			return nil, nil, fmt.Errorf("exceeded max number of consumed outputs 256")
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

	if availableTokens < amount {
		return nil, nil, fmt.Errorf("not enough tokens in address %s: needed %d, got %d",
			par.SenderAddress.String(), par.Amount, availableTokens)
	}
	txb := NewTransactionBuilder()
	for _, o := range consumedOuts {
		if _, err := txb.ConsumeOutput(o.Output, o.ID); err != nil {
			return nil, nil, err
		}
	}
	out := NewOutput().
		WithAmount(amount).
		WithTimestamp(ts).
		WithLockConstraint(par.Lock)
	if par.AddSender {
		if _, err := out.PushConstraint(library.NewSenderAddressED25519(par.SenderAddress).Bytes()); err != nil {
			return nil, nil, err
		}
	}

	for _, constr := range par.AddConstraints {
		if _, err := out.PushConstraint(constr); err != nil {
			return nil, nil, err
		}
	}
	if _, err := txb.ProduceOutput(out); err != nil {
		return nil, nil, err
	}
	if availableTokens > amount {
		reminderOut := NewOutput().
			WithAmount(availableTokens - amount).
			WithTimestamp(ts).
			WithLockConstraint(par.SenderAddress)
		if _, err := txb.ProduceOutput(reminderOut); err != nil {
			return nil, nil, err
		}
	}
	txb.Transaction.Timestamp = ts
	txb.Transaction.InputCommitment = txb.InputCommitment()
	txb.SignED25519(par.SenderPrivateKey)

	for i := range consumedOuts {
		if i == 0 {
			txb.PutSignatureUnlock(0, library.ConstraintIndexLock)
		} else {
			// always referencing the 0 output
			err := txb.PutUnlockReference(byte(i), library.ConstraintIndexLock, 0)
			easyfl.AssertNoError(err)
		}
	}
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

func ParseChainConstraints(outs []*ledger.OutputDataWithID) ([]*ledger.OutputDataWithChainID, error) {
	ret := make([]*ledger.OutputDataWithChainID, 0)
	err := ForEachOutput(outs, func(o *Output, odata *ledger.OutputDataWithID) bool {
		ch, constraintIndex := o.ChainConstraint()
		if constraintIndex == 0xff {
			return true
		}
		d := &ledger.OutputDataWithChainID{
			OutputDataWithID:           *odata,
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
