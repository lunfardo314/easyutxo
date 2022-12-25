package state

import (
	"time"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraints"
	"github.com/lunfardo314/easyutxo/ledger/indexer"
	"github.com/lunfardo314/unitrie/common"
	"github.com/lunfardo314/unitrie/immutable"
	"github.com/lunfardo314/unitrie/models/trie_blake2b"
)

type (
	// Updatable is an updatable ledger state, with the particular root
	// Suitable for chained updates
	Updatable struct {
		store ledger.StateStore
		root  common.VCommitment
	}

	// Readable is a read-only ledger state, with the particular root
	Readable struct {
		trie *immutable.TrieReader
	}
)

// commitment model singleton

var commitmentModel = trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256)

// MustInitLedgerState initializes origin ledger state in the empty store
func MustInitLedgerState(store common.KVWriter, identity []byte, genesisAddr constraints.AddressED25519, initialSupply uint64) common.VCommitment {
	storeTmp := common.NewInMemoryKVStore()
	emptyRoot := immutable.MustInitRoot(storeTmp, commitmentModel, identity)
	trie, err := immutable.NewTrieChained(commitmentModel, storeTmp, emptyRoot)
	easyfl.AssertNoError(err)

	outBytes, oid := genesisOutput(genesisAddr, initialSupply, uint32(time.Now().Unix()))
	trie.Update(oid[:], outBytes)
	trie = trie.CommitChained()
	common.CopyAll(store, storeTmp)
	return trie.Root()
}

// NewReadable creates read-only ledger state with the given root
func NewReadable(store common.KVReader, root common.VCommitment) (*Readable, error) {
	trie, err := immutable.NewTrieReader(commitmentModel, store, root)
	if err != nil {
		return nil, err
	}
	return &Readable{trie}, nil
}

// NewUpdatable creates updatable state with the given root. After updated, the root changes.
// Suitable for chained updates of the ledger state
func NewUpdatable(store ledger.StateStore, root common.VCommitment) (*Updatable, error) {
	_, err := immutable.NewTrieReader(commitmentModel, store, root)
	if err != nil {
		return nil, err
	}
	return &Updatable{
		root:  root.Clone(),
		store: store,
	}, nil
}

// genesisOutput creates genesis output which contains initialSupply and timestamp. The genesis outputID is all-0
func genesisOutput(genesisAddr constraints.AddressED25519, initialSupply uint64, ts uint32) ([]byte, ledger.OutputID) {
	easyfl.Assert(initialSupply > 0, "initialSupply > 0")
	amount := constraints.NewAmount(initialSupply)
	timestamp := constraints.NewTimestamp(ts)
	ret := lazyslice.EmptyArray()
	ret.Push(amount.Bytes())
	ret.Push(timestamp.Bytes())
	ret.Push(genesisAddr.Bytes())
	return ret.Bytes(), ledger.OutputID{}
}

func (u *Updatable) Readable() *Readable {
	trie, err := immutable.NewTrieReader(commitmentModel, u.store, u.root)
	common.AssertNoError(err)
	return &Readable{
		trie: trie,
	}
}

func (r *Readable) GetUTXO(oid *ledger.OutputID) ([]byte, bool) {
	ret := r.trie.Get(oid.Bytes())
	if len(ret) == 0 {
		return nil, false
	}
	return ret, true
}

func (r *Readable) HasTransaction(txid *ledger.TransactionID) bool {
	ret := false
	r.trie.Iterator(txid.Bytes()).IterateKeys(func(_ []byte) bool {
		ret = true
		return false
	})
	return ret
}

// Root return the current root
func (u *Updatable) Root() common.VCommitment {
	return u.root
}

// Update updates/mutates the ledger state by transaction
func (u *Updatable) Update(txBytes []byte, traceOption ...int) ([]*indexer.Command, error) {
	ctx, err := TransactionContextFromTransferableBytes(txBytes, u.Readable(), traceOption...)
	if err != nil {
		return nil, err
	}
	trie, err := immutable.NewTrieUpdatable(commitmentModel, u.store, u.root)
	if err != nil {
		return nil, err
	}
	indexerUpdate, err := updateTrieMulti(trie, []*TransactionContext{ctx})
	if err != nil {
		return nil, err
	}
	batch := u.store.BatchedWriter()
	u.root = trie.Commit(batch)
	return indexerUpdate, batch.Commit()
}

// UpdateMulti updates/mutates the ledger state by transaction
func updateTrieMulti(trie *immutable.TrieUpdatable, txCtx []*TransactionContext) ([]*indexer.Command, error) {
	indexerUpdate := make([]*indexer.Command, 0)
	for _, ctx := range txCtx {
		iu, err := updateTrie(trie, ctx)
		if err != nil {
			return nil, err
		}
		indexerUpdate = append(indexerUpdate, iu...)
	}
	return indexerUpdate, nil
}

// updateTrie updates trie from transaction without committing
func updateTrie(trie *immutable.TrieUpdatable, ctx *TransactionContext) ([]*indexer.Command, error) {
	indexerUpdate, err := ctx.Validate()
	if err != nil {
		return nil, err
	}

	// delete consumed outputs from the ledger and from accounts
	ctx.ForEachInputID(func(idx byte, oid *ledger.OutputID) bool {
		trie.Update(oid[:], nil)
		return true
	})

	// add new outputs to the ledger and to accounts
	txID := ctx.TransactionID()
	ctx.tree.ForEach(func(idx byte, outputData []byte) bool {
		oid := ledger.NewOutputID(txID, idx)
		trie.Update(oid[:], outputData)
		return true
	}, Path(constraints.TransactionBranch, constraints.TxOutputs))

	return indexerUpdate, nil
}
