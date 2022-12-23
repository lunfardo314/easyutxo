package state

import (
	"time"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/indexer"
	"github.com/lunfardo314/easyutxo/ledger/library"
	"github.com/lunfardo314/unitrie/common"
	"github.com/lunfardo314/unitrie/immutable"
	"github.com/lunfardo314/unitrie/models/trie_blake2b"
)

// Updatable is a ledger state, with the particular root
type (
	Updatable struct {
		store ledger.StateStore
		root  common.VCommitment
	}

	Readable struct {
		trie *immutable.TrieReader
	}
)

// commitment model singleton

var commitmentModel = trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256)

// MustInitLedgerState initializes origin ledger state in the empty store
func MustInitLedgerState(store common.KVWriter, identity []byte, genesisAddr library.AddressED25519, initialSupply uint64) common.VCommitment {
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

func NewReadable(store common.KVReader, root common.VCommitment) (*Readable, error) {
	trie, err := immutable.NewTrieReader(commitmentModel, store, root)
	if err != nil {
		return nil, err
	}
	return &Readable{trie}, nil
}

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

func genesisOutput(genesisAddr library.AddressED25519, initialSupply uint64, ts uint32) ([]byte, ledger.OutputID) {
	easyfl.Assert(initialSupply > 0, "initialSupply > 0")
	amount := library.NewAmount(initialSupply)
	timestamp := library.NewTimestamp(ts)
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

const clearCacheSizeAfter = 1000

func (u *Updatable) AddTransaction(txBytes []byte, traceOption ...int) ([]*indexer.Command, error) {
	stateReader, err := NewReadable(u.store, u.root)
	if err != nil {
		return nil, err
	}
	ctx, err := ValidationContextFromTransaction(txBytes, stateReader, traceOption...)
	if err != nil {
		return nil, err
	}
	indexerUpdate, err := ctx.Validate()
	if err != nil {
		return nil, err
	}
	return indexerUpdate, u.updateLedger(ctx)
}

func (u *Updatable) updateLedger(ctx *ValidationContext) error {
	trie, err := immutable.NewTrieUpdatable(commitmentModel, u.store, u.root)
	if err != nil {
		return err
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
	}, Path(library.TransactionBranch, library.TxOutputs))

	batch := u.store.BatchedWriter()
	u.root = trie.Commit(batch)
	return batch.Commit()
}
