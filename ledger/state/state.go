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
type Updatable struct {
	store ledger.StateStore
	trie  *immutable.TrieUpdatable
}

type Readable struct {
	trie *immutable.TrieReader
}

// commitment model singleton

var commitmentModel = trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256)

// MustInitLedgerState initializes origin ledger state in the empty store
func MustInitLedgerState(store common.KVWriter, identity []byte, genesisAddr library.AddressED25519, initialSupply uint64) common.VCommitment {
	storeTmp := common.NewInMemoryKVStore()
	emptyRoot := immutable.MustInitRoot(storeTmp, commitmentModel, identity)
	trie, err := immutable.NewTrieUpdatable(commitmentModel, storeTmp, emptyRoot)
	easyfl.AssertNoError(err)

	outBytes, oid := genesisOutput(genesisAddr, initialSupply, uint32(time.Now().Unix()))
	trie.Update(oid[:], outBytes)
	ret := trie.Commit(storeTmp)
	common.CopyAll(store, storeTmp)
	return ret
}

func NewReadable(store common.KVReader, root common.VCommitment) (*Readable, error) {
	trie, err := immutable.NewTrieReader(commitmentModel, store, root)
	if err != nil {
		return nil, err
	}
	return &Readable{trie}, nil
}

func NewUpdatable(store ledger.StateStore, root common.VCommitment) (*Updatable, error) {
	trie, err := immutable.NewTrieUpdatable(commitmentModel, store, root)
	if err != nil {
		return nil, err
	}
	return &Updatable{
		store: store,
		trie:  trie,
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
	return &Readable{
		trie: u.trie.TrieReader,
	}
}

func (u *Updatable) AddTransaction(txBytes []byte, traceOption ...int) (common.VCommitment, []*indexer.Command, error) {
	ctx, err := ValidationContextFromTransaction(txBytes, u.Readable(), traceOption...)
	if err != nil {
		return nil, nil, err
	}
	indexerUpdate, err := ctx.Validate()
	if err != nil {
		return nil, nil, err
	}
	root, err := u.updateLedger(ctx)
	return root, indexerUpdate, err
}

func (r *Readable) GetUTXO(oid *ledger.OutputID) ([]byte, bool) {
	ret := r.trie.Get(oid.Bytes())
	if len(ret) == 0 {
		return nil, false
	}
	return ret, true
}

func (u *Updatable) updateLedger(ctx *ValidationContext) (common.VCommitment, error) {
	// delete consumed outputs from the ledger and from accounts
	ctx.ForEachInputID(func(idx byte, oid *ledger.OutputID) bool {
		u.trie.Update(oid[:], nil)
		return true
	})
	// add new outputs to the ledger and to accounts
	txID := ctx.TransactionID()
	ctx.tree.ForEach(func(idx byte, outputData []byte) bool {
		oid := ledger.NewOutputID(txID, idx)
		u.trie.Update(oid[:], outputData)
		return true
	}, Path(library.TransactionBranch, library.TxOutputs))

	return u.trie.Persist(u.store.BatchedWriter())
}
