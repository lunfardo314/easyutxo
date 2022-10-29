package ledger

import (
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"sort"
	"time"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger/constraint"
	"golang.org/x/crypto/blake2b"
)

const (
	TransactionIDLength = 32
)

// Transaction tree
const (
	TxUnlockParamsBranch = byte(iota)
	TxInputIDsBranch
	TxOutputBranch
	TxTimestamp
	TxInputCommitment
	TxTreeIndexMax
)

type (
	TransactionID [TransactionIDLength]byte

	TransactionBuilder struct {
		ConsumedOutputs []*Output
		Transaction     *transaction
	}

	transaction struct {
		InputIDs        []*OutputID
		Outputs         []*Output
		UnlockBlocks    []*UnlockParams
		Timestamp       uint32
		InputCommitment [32]byte
	}

	UnlockParams struct {
		array *lazyslice.Array
	}
)

func (txid *TransactionID) String() string {
	return easyfl.Fmt(txid[:])
}

func NewTransactionBuilder() *TransactionBuilder {
	return &TransactionBuilder{
		ConsumedOutputs: make([]*Output, 0),
		Transaction: &transaction{
			InputIDs:        make([]*OutputID, 0),
			Outputs:         make([]*Output, 0),
			UnlockBlocks:    make([]*UnlockParams, 0),
			Timestamp:       0,
			InputCommitment: [32]byte{},
		},
	}
}

func (ctx *TransactionBuilder) NumInputs() int {
	ret := len(ctx.ConsumedOutputs)
	easyfl.Assert(ret == len(ctx.Transaction.InputIDs), "ret==len(ctx.Transaction.InputIDs)")
	return ret
}

func (ctx *TransactionBuilder) NumOutputs() int {
	return len(ctx.Transaction.Outputs)
}

func (ctx *TransactionBuilder) ConsumeOutput(out *Output, oid OutputID) (byte, error) {
	if ctx.NumInputs() >= 256 {
		return 0, fmt.Errorf("too many consumed outputs")
	}
	ctx.ConsumedOutputs = append(ctx.ConsumedOutputs, out)
	ctx.Transaction.InputIDs = append(ctx.Transaction.InputIDs, &oid)
	ctx.Transaction.UnlockBlocks = append(ctx.Transaction.UnlockBlocks, NewUnlockBlock())

	return byte(len(ctx.ConsumedOutputs) - 1), nil
}

func (ctx *TransactionBuilder) UnlockBlock(idx byte) *UnlockParams {
	return ctx.Transaction.UnlockBlocks[idx]
}

func (ctx *TransactionBuilder) ProduceOutput(out *Output) (byte, error) {
	if ctx.NumOutputs() >= 256 {
		return 0, fmt.Errorf("too many produced outputs")
	}
	ctx.Transaction.Outputs = append(ctx.Transaction.Outputs, out)
	return byte(len(ctx.Transaction.Outputs) - 1), nil
}

func (ctx *TransactionBuilder) InputCommitment() [32]byte {
	arr := lazyslice.EmptyArray(256)
	for _, o := range ctx.ConsumedOutputs {
		b := o.Bytes()
		arr.Push(b)
	}
	return blake2b.Sum256(arr.Bytes())
}

func (tx *transaction) ToArray() *lazyslice.Array {
	unlockBlocks := lazyslice.EmptyArray(256)
	inputIDs := lazyslice.EmptyArray(256)
	outputs := lazyslice.EmptyArray(256)

	for _, b := range tx.UnlockBlocks {
		unlockBlocks.Push(b.Bytes())
	}
	for _, oid := range tx.InputIDs {
		inputIDs.Push(oid[:])
	}
	for _, o := range tx.Outputs {
		outputs.Push(o.Bytes())
	}

	var ts [4]byte
	binary.BigEndian.PutUint32(ts[:], tx.Timestamp)
	ret := lazyslice.MakeArray(
		unlockBlocks,          // TxUnlockParamsBranch = 0
		inputIDs,              // TxInputIDsBranch = 1
		outputs,               // TxOutputBranch = 2
		ts[:],                 // TxTimestamp = 3
		tx.InputCommitment[:], // TxInputCommitment = 4
	)

	return ret
}

func (tx *transaction) Bytes() []byte {
	return tx.ToArray().Bytes()
}

func (tx *transaction) ID() TransactionID {
	return blake2b.Sum256(tx.Bytes())
}

func (tx *transaction) EssenceBytes() []byte {
	arr := tx.ToArray()
	return easyfl.Concat(
		arr.At(int(TxInputIDsBranch)),
		arr.At(int(TxOutputBranch)),
		arr.At(int(TxInputCommitment)),
	)
}

type ED25519TransferInputs struct {
	SenderPrivateKey ed25519.PrivateKey
	SenderPublicKey  ed25519.PublicKey
	SenderAddress    constraint.AddressED25519
	Outputs          []*OutputWithID
	Timestamp        uint32 // takes time.Now() if 0
	Lock             constraint.Lock
	Amount           uint64
	AdjustToMinimum  bool
	AddConstraints   [][]byte
}

func MakeED25519TransferInputs(senderKey ed25519.PrivateKey, state StateAccess, desc ...bool) (*ED25519TransferInputs, error) {
	sourcePubKey := senderKey.Public().(ed25519.PublicKey)
	sourceAddr := constraint.AddressED25519FromPublicKey(sourcePubKey)
	ret := &ED25519TransferInputs{
		SenderPrivateKey: senderKey,
		SenderPublicKey:  sourcePubKey,
		SenderAddress:    sourceAddr,
		Timestamp:        uint32(time.Now().Unix()),
		AddConstraints:   make([][]byte, 0),
	}
	var err error
	ret.Outputs, err = state.GetUTXOsForAddress(sourceAddr)
	if err != nil {
		return nil, err
	}
	if len(ret.Outputs) == 0 {
		return nil, fmt.Errorf("empty account %s", easyfl.Fmt(sourceAddr))
	}
	descending := len(desc) > 0 && desc[0]
	if descending {
		sort.Slice(ret.Outputs, func(i, j int) bool {
			return ret.Outputs[i].Output.Amount() > ret.Outputs[j].Output.Amount()
		})
	} else {
		sort.Slice(ret.Outputs, func(i, j int) bool {
			return ret.Outputs[i].Output.Amount() < ret.Outputs[j].Output.Amount()
		})
	}
	return ret, nil
}

func (t *ED25519TransferInputs) WithTimestamp(ts uint32) *ED25519TransferInputs {
	t.Timestamp = ts
	return t
}

func (t *ED25519TransferInputs) WithTargetLock(lock constraint.Lock) *ED25519TransferInputs {
	t.Lock = lock
	return t
}

func (t *ED25519TransferInputs) WithAmount(amount uint64, adjustToMinimum ...bool) *ED25519TransferInputs {
	t.Amount = amount
	t.AdjustToMinimum = len(adjustToMinimum) > 0 && adjustToMinimum[0]
	return t
}

func (t *ED25519TransferInputs) WithConstraint(constr constraint.Constraint) *ED25519TransferInputs {
	t.AddConstraints = append(t.AddConstraints, constr.Bytes())
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
	minimumDeposit := MinimumStorageDeposit(uint32(len(outTentative.Bytes())), 0)
	if t.Amount < minimumDeposit {
		return minimumDeposit
	}
	return t.Amount
}

func MakeTransferTransaction(par *ED25519TransferInputs) ([]byte, error) {
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
			return nil, fmt.Errorf("exceeded max number of consumed outputs 256")
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
		return nil, fmt.Errorf("not enough tokens in address %s: needed %d, got %d",
			par.SenderAddress.String(), par.Amount, availableTokens)
	}
	ctx := NewTransactionBuilder()
	for _, o := range consumedOuts {
		if _, err := ctx.ConsumeOutput(o.Output, o.ID); err != nil {
			return nil, err
		}
	}
	out := NewOutput().
		WithAmount(amount).
		WithTimestamp(ts).
		WithLockConstraint(par.Lock)

	for _, constr := range par.AddConstraints {
		if _, err := out.PushConstraint(constr); err != nil {
			return nil, err
		}
	}
	if _, err := ctx.ProduceOutput(out); err != nil {
		return nil, err
	}
	if availableTokens > amount {
		reminderOut := NewOutput().
			WithAmount(availableTokens - amount).
			WithTimestamp(ts).
			WithLockConstraint(par.SenderAddress)
		if _, err := ctx.ProduceOutput(reminderOut); err != nil {
			return nil, err
		}
	}
	ctx.Transaction.Timestamp = ts
	ctx.Transaction.InputCommitment = ctx.InputCommitment()

	unlockDataRef := constraint.UnlockParamsByReference(0)
	for i := range consumedOuts {
		if i == 0 {
			unlockData := constraint.UnlockParamsBySignatureED25519(ctx.Transaction.EssenceBytes(), par.SenderPrivateKey)
			ctx.UnlockBlock(0).PutUnlockParams(unlockData, OutputBlockLock)
			continue
		}
		ctx.UnlockBlock(byte(i)).PutUnlockParams(unlockDataRef, OutputBlockLock)
	}
	return ctx.Transaction.Bytes(), nil
}
