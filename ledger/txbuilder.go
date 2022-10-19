package ledger

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"golang.org/x/crypto/blake2b"
)

const (
	TransactionIDLength = 32
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
		UnlockBlocks    []DataBlockRaw
		Timestamp       uint32
		InputCommitment [32]byte
	}

	DataBlockRaw []byte
)

func (txid *TransactionID) String() string {
	return hex.EncodeToString(txid[:])
}

func (ctx *TransactionContext) NumInputs() int {
	ret := len(ctx.ConsumedOutputs)
	common.Assert(ret == len(ctx.Transaction.InputIDs), "ret==len(ctx.Transaction.InputIDs)")
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
	return byte(len(ctx.ConsumedOutputs) - 1), nil
}

func (ctx *TransactionContext) ProduceOutput(out *Output) (byte, error) {
	if ctx.NumOutputs() >= 256 {
		return 0, fmt.Errorf("too many produced outputs")
	}
	ctx.Transaction.Outputs = append(ctx.Transaction.Outputs, out)
	return byte(len(ctx.Transaction.Outputs) - 1), nil
}

func (ctx *TransactionContext) InputCommitment() [32]byte {
	var buf bytes.Buffer

	for _, o := range ctx.ConsumedOutputs {
		buf.Write(o.Bytes())
	}
	return blake2b.Sum256(buf.Bytes())
}

func (tx *Transaction) ToArray() *lazyslice.Array {
	unlockBlocks := lazyslice.EmptyArray(256)
	inputIDs := lazyslice.EmptyArray(256)
	outputs := lazyslice.EmptyArray(256)

	for _, b := range tx.UnlockBlocks {
		unlockBlocks.Push(b)
	}
	for _, oid := range tx.InputIDs {
		inputIDs.Push(oid[:])
	}
	for _, o := range tx.Outputs {
		outputs.Push(o.Bytes())
	}

	ret := lazyslice.EmptyArray(256)
	ret.PushEmptyElements(int(TxTreeIndexMax))
	ret.PutAtIdx(TxUnlockParamsBranch, unlockBlocks.Bytes())
	ret.PutAtIdx(TxInputIDsBranch, inputIDs.Bytes())
	ret.PutAtIdx(TxOutputBranch, outputs.Bytes())
	var ts [4]byte
	binary.BigEndian.PutUint32(ts[:], tx.Timestamp)
	ret.PutAtIdx(TxTimestamp, ts[:])
	ret.PutAtIdx(TxInputCommitment, tx.InputCommitment[:])

	return ret
}

func (tx *Transaction) Bytes() []byte {
	return tx.ToArray().Bytes()
}

func (tx *Transaction) ID() TransactionID {
	return blake2b.Sum256(tx.Bytes())
}
