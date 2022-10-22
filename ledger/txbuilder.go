package ledger

import (
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
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
		InputIDs        []OutputID
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
	return hex.EncodeToString(txid[:])
}

func NewTransactionContext() *TransactionContext {
	return &TransactionContext{
		ConsumedOutputs: make([]*Output, 0),
		Transaction: &Transaction{
			InputIDs:        make([]OutputID, 0),
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
	ctx.Transaction.InputIDs = append(ctx.Transaction.InputIDs, oid)
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
		arr.Push(o.Bytes())
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

func MakeTransferTransaction(ledger StateAccess, privKey ed25519.PrivateKey, targetAddress Address, amount uint64) ([]byte, error) {
	sourcePubKey := privKey.Public().(ed25519.PublicKey)
	sourceAddr := AddressFromED25519PubKey(sourcePubKey)
	outs, err := ledger.GetUTXOsForAddress(sourceAddr)
	if err != nil {
		return nil, err
	}
	sort.Slice(outs, func(i, j int) bool {
		return outs[i].Output.Amount < outs[j].Output.Amount
	})
	consumedOuts := outs[:0]
	availableTokens := uint64(0)
	ts := uint32(time.Now().Unix())
	for _, o := range outs {
		consumedOuts = append(consumedOuts, o)
		if o.Output.Timestamp >= ts {
			ts = o.Output.Timestamp + 1
		}
		availableTokens += o.Output.Amount
		if availableTokens >= amount {
			break
		}
	}
	if availableTokens < amount {
		return nil, fmt.Errorf("not enough tokens in address %s: needed %d, got %d",
			sourceAddr.String(), amount, availableTokens)
	}
	ctx := NewTransactionContext()
	for _, o := range consumedOuts {
		if _, err = ctx.ConsumeOutput(o.Output, o.ID); err != nil {
			return nil, err
		}
	}
	out := NewOutput(amount, ts, targetAddress)
	if _, err = ctx.ProduceOutput(out); err != nil {
		return nil, err
	}
	if availableTokens > amount {
		reminderOut := NewOutput(availableTokens-amount, ts, sourceAddr)
		if _, err = ctx.ProduceOutput(reminderOut); err != nil {
			return nil, err
		}
	}
	ctx.Transaction.Timestamp = ts
	ctx.Transaction.InputCommitment = ctx.InputCommitment()

	unlockDataRef := UnlockDataByReference(0)
	for i := range consumedOuts {
		if i == 0 {
			unlockData := UnlockDataBySignatureED25519(ctx.Transaction.EssenceBytes(), privKey)
			ctx.UnlockBlock(0).PutUnlockParams(unlockData, OutputBlockAddress)
			continue
		}
		ctx.UnlockBlock(byte(i)).PutUnlockParams(unlockDataRef, OutputBlockAddress)
	}
	return ctx.Transaction.Bytes(), nil
}
