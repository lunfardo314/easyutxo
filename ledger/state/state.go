package state

import (
	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraints"
	"github.com/lunfardo314/easyutxo/ledger/indexer"
	"github.com/lunfardo314/easyutxo/util/lazyslice"
	"github.com/lunfardo314/unitrie/common"
	"github.com/lunfardo314/unitrie/immutable"
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

	InitLedgerStateParams struct {
		Store                 common.KVWriter
		Identity              []byte
		InitialSupply         uint64
		GenesisAddress        constraints.AddressED25519
		Timestamp             uint32
		FinalityGadgetAddress constraints.AddressED25519 // FG output is created when not nil
	}
)

// InitLedgerState initializes origin ledger state in the empty store
// If FinalityGadgetAddress != nil, it also creates chain output for the finality gadget
// Returns root commitment to the genesis ledger state
func InitLedgerState(par InitLedgerStateParams) common.VCommitment {
	storeTmp := common.NewInMemoryKVStore()
	emptyRoot := immutable.MustInitRoot(storeTmp, ledger.CommitmentModel, par.Identity)

	trie, err := immutable.NewTrieChained(ledger.CommitmentModel, storeTmp, emptyRoot)
	easyfl.AssertNoError(err)

	genesisAmount := par.InitialSupply
	if len(par.FinalityGadgetAddress) > 0 {
		genesisAmount -= ledger.MilestoneDeposit
	}
	trie.Update(ledger.GenesisOutputID[:], genesisOutput(genesisAmount, par.GenesisAddress, par.Timestamp))

	if len(par.FinalityGadgetAddress) > 0 {
		trie.Update(ledger.GenesisMilestoneOutputID[:], genesisMilestoneOutput(par.FinalityGadgetAddress, par.Timestamp))
	}

	trie = trie.CommitChained()
	common.CopyAll(par.Store, storeTmp)
	return trie.Root()
}

func genesisOutput(initialSupply uint64, address constraints.AddressED25519, ts uint32) []byte {
	amount := constraints.NewAmount(initialSupply)
	timestamp := constraints.NewTimestamp(ts)
	ret := lazyslice.EmptyArray()
	ret.Push(amount.Bytes())
	ret.Push(timestamp.Bytes())
	ret.Push(address.Bytes())
	return ret.Bytes()
}

func genesisMilestoneOutput(address constraints.AddressED25519, ts uint32) []byte {
	amount := constraints.NewAmount(ledger.MilestoneDeposit) //  - g.MilestoneDeposit)
	timestamp := constraints.NewTimestamp(ts)
	ret := lazyslice.EmptyArray()
	ret.Push(amount.Bytes())
	ret.Push(timestamp.Bytes())
	ret.Push(address.Bytes())
	ret.Push(constraints.NewChainConstraint(ledger.MilestoneChainID, 0, 0, 0).Bytes())
	ret.Push(constraints.NewStateIndex(4, 0).Bytes())
	return ret.Bytes()
}

// NewReadable creates read-only ledger state with the given root
func NewReadable(store common.KVReader, root common.VCommitment, clearCacheAtSize ...int) (*Readable, error) {
	trie, err := immutable.NewTrieReader(ledger.CommitmentModel, store, root, clearCacheAtSize...)
	if err != nil {
		return nil, err
	}
	return &Readable{trie}, nil
}

// NewUpdatable creates updatable state with the given root. After updated, the root changes.
// Suitable for chained updates of the ledger state
func NewUpdatable(store ledger.StateStore, root common.VCommitment) (*Updatable, error) {
	_, err := immutable.NewTrieReader(ledger.CommitmentModel, store, root)
	if err != nil {
		return nil, err
	}
	return &Updatable{
		root:  root.Clone(),
		store: store,
	}, nil
}

func (u *Updatable) Readable() *Readable {
	trie, err := immutable.NewTrieReader(ledger.CommitmentModel, u.store, u.root)
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
	trie, err := immutable.NewTrieUpdatable(ledger.CommitmentModel, u.store, u.root)
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
