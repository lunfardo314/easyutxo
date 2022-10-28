package ledger

import (
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"math"
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

	TransactionContext struct {
		ConsumedOutputs []*Output
		Transaction     *Transaction
	}

	Transaction struct {
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

func NewTransactionContext() *TransactionContext {
	return &TransactionContext{
		ConsumedOutputs: make([]*Output, 0),
		Transaction: &Transaction{
			InputIDs:        make([]*OutputID, 0),
			Outputs:         make([]*Output, 0),
			UnlockBlocks:    make([]*UnlockParams, 0),
			Timestamp:       0,
			InputCommitment: [32]byte{},
		},
	}
}

func (ctx *TransactionContext) NumInputs() int {
	ret := len(ctx.ConsumedOutputs)
	easyfl.Assert(ret == len(ctx.Transaction.InputIDs), "ret==len(ctx.Transaction.InputIDs)")
	return ret
}

func (ctx *TransactionContext) NumOutputs() int {
	return len(ctx.Transaction.Outputs)
}

func (ctx *TransactionContext) ConsumeOutput(out *Output, oid OutputID) (byte, error) {
	if ctx.NumInputs() >= 256 {
		return 0, fmt.Errorf("too many consumed outputs")
	}
	ctx.ConsumedOutputs = append(ctx.ConsumedOutputs, out)
	ctx.Transaction.InputIDs = append(ctx.Transaction.InputIDs, &oid)
	ctx.Transaction.UnlockBlocks = append(ctx.Transaction.UnlockBlocks, NewUnlockBlock())

	return byte(len(ctx.ConsumedOutputs) - 1), nil
}

func (ctx *TransactionContext) UnlockBlock(idx byte) *UnlockParams {
	return ctx.Transaction.UnlockBlocks[idx]
}

func (ctx *TransactionContext) ProduceOutput(out *Output) (byte, error) {
	if ctx.NumOutputs() >= 256 {
		return 0, fmt.Errorf("too many produced outputs")
	}
	ctx.Transaction.Outputs = append(ctx.Transaction.Outputs, out)
	return byte(len(ctx.Transaction.Outputs) - 1), nil
}

func (ctx *TransactionContext) InputCommitment() [32]byte {
	arr := lazyslice.EmptyArray(256)
	for _, o := range ctx.ConsumedOutputs {
		b := o.Bytes()
		arr.Push(b)
	}
	return blake2b.Sum256(arr.Bytes())
}

func (tx *Transaction) ToArray() *lazyslice.Array {
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

func (tx *Transaction) Bytes() []byte {
	return tx.ToArray().Bytes()
}

func (tx *Transaction) ID() TransactionID {
	return blake2b.Sum256(tx.Bytes())
}

func (tx *Transaction) EssenceBytes() []byte {
	arr := tx.ToArray()
	return easyfl.Concat(
		arr.At(int(TxInputIDsBranch)),
		arr.At(int(TxOutputBranch)),
		arr.At(int(TxInputCommitment)),
	)
}

type TransferTransactionParams struct {
	Timestamp         uint32 // takes time.Now() if 0
	SenderKey         ed25519.PrivateKey
	TargetAddress     Constraint
	Amount            uint64
	AdjustToMinimum   bool
	DescendingOutputs bool
	AddSender         bool
	AddTimeLockForSec uint32 // add seconds to timestamp for time lock condition. No effect if 0
}

func MakeTransferTransaction(ledger StateAccess, par TransferTransactionParams) ([]byte, error) {
	sourcePubKey := par.SenderKey.Public().(ed25519.PublicKey)
	sourceAddr := constraint.AddressED25519SigLock(sourcePubKey)
	outs, err := ledger.GetUTXOsForAddress(sourceAddr)
	if err != nil {
		return nil, err
	}

	ts := uint32(time.Now().Unix())
	if par.Timestamp > 0 {
		ts = par.Timestamp
	}
	amount := AdjustedAmount(&par)

	if par.DescendingOutputs {
		sort.Slice(outs, func(i, j int) bool {
			return outs[i].Output.Amount() > outs[j].Output.Amount()
		})
	} else {
		sort.Slice(outs, func(i, j int) bool {
			return outs[i].Output.Amount() < outs[j].Output.Amount()
		})
	}
	consumedOuts := outs[:0]
	availableTokens := uint64(0)
	numConsumedOutputs := 0

	for _, o := range outs {
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
			constraint.SigLockToString(sourceAddr), par.Amount, availableTokens)
	}
	ctx := NewTransactionContext()
	for _, o := range consumedOuts {
		if _, err = ctx.ConsumeOutput(o.Output, o.ID); err != nil {
			return nil, err
		}
	}
	out := NewOutput().
		WithAmount(amount).
		WithTimestamp(ts).
		WithLockConstraint(par.TargetAddress)

	if par.AddSender {
		// add sender constraint
		if _, err = out.PushConstraint(constraint.Sender(sourceAddr, 0)); err != nil {
			return nil, err
		}
	}
	if par.AddTimeLockForSec > 0 {
		// add time lock constraint
		if par.AddTimeLockForSec > math.MaxUint32-ts {
			return nil, fmt.Errorf("wrong timelock delta %d", par.AddTimeLockForSec)
		}
		if _, err = out.PushConstraint(constraint.Timelock(ts + par.AddTimeLockForSec)); err != nil {
			return nil, err
		}
	}
	if _, err = ctx.ProduceOutput(out); err != nil {
		return nil, err
	}
	if availableTokens > amount {
		reminderOut := NewOutput().
			WithAmount(availableTokens - amount).
			WithTimestamp(ts).
			WithLockConstraint(sourceAddr)
		if _, err = ctx.ProduceOutput(reminderOut); err != nil {
			return nil, err
		}
	}
	ctx.Transaction.Timestamp = ts
	ctx.Transaction.InputCommitment = ctx.InputCommitment()

	unlockDataRef := constraint.UnlockParamsByReference(0)
	for i := range consumedOuts {
		if i == 0 {
			unlockData := constraint.UnlockParamsBySignatureED25519(ctx.Transaction.EssenceBytes(), par.SenderKey)
			ctx.UnlockBlock(0).PutUnlockParams(unlockData, OutputBlockLock)
			continue
		}
		ctx.UnlockBlock(byte(i)).PutUnlockParams(unlockDataRef, OutputBlockLock)
	}
	return ctx.Transaction.Bytes(), nil
}

// AdjustedAmount adjust amount to minimum storage deposit requirements
func AdjustedAmount(par *TransferTransactionParams) uint64 {
	if !par.AdjustToMinimum {
		// not adjust. Will render wrong transaction if not enough tokens
		return par.Amount
	}
	ts := uint32(0)

	outTentative := NewOutput()
	outTentative.WithAmount(par.Amount)
	outTentative.WithTimestamp(ts)
	outTentative.WithLockConstraint(par.TargetAddress)
	if par.AddSender {
		_, err := outTentative.PushConstraint(constraint.Sender(constraint.AddressED25519SigLockNull(), 0))
		easyfl.AssertNoError(err)
	}
	if par.AddTimeLockForSec > 0 {
		_, err := outTentative.PushConstraint(constraint.Timelock(par.AddTimeLockForSec))
		easyfl.AssertNoError(err)
	}
	minimumDeposit := MinimumStorageDeposit(uint32(len(outTentative.Bytes())), 0)
	if par.Amount < minimumDeposit {
		return minimumDeposit
	}
	return par.Amount
}
