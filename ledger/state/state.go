package state

import (
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraints"
	"github.com/lunfardo314/easyutxo/ledger/indexer"
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
)

// InitLedgerState initializes origin ledger state in the empty store
func InitLedgerState(store common.KVWriter, g *ledger.GenesisDataStruct) common.VCommitment {
	storeTmp := common.NewInMemoryKVStore()
	emptyRoot := immutable.MustInitRoot(storeTmp, ledger.CommitmentModel, g.StateIdentity)

	trie, err := immutable.NewTrieChained(ledger.CommitmentModel, storeTmp, emptyRoot)
	easyfl.AssertNoError(err)

	trie.Update(g.OutputID[:], genesisOutput(g))
	trie = trie.CommitChained()

	genesisRoot := trie.Root()
	trie.Update(g.MilestoneOutputID[:], genesisMilestoneOutput(g, genesisRoot))
	trie = trie.CommitChained()

	common.CopyAll(store, storeTmp)
	return genesisRoot
}

func genesisOutput(g *ledger.GenesisDataStruct) []byte {
	amount := constraints.NewAmount(g.InitialSupply) //  - g.MilestoneDeposit)
	timestamp := constraints.NewTimestamp(g.Timestamp)
	ret := lazyslice.EmptyArray()
	ret.Push(amount.Bytes())
	ret.Push(timestamp.Bytes())
	ret.Push(g.Address.Bytes())
	return ret.Bytes()
}

func genesisMilestoneOutput(g *ledger.GenesisDataStruct, genesisStateCommitment common.VCommitment) []byte {
	amount := constraints.NewAmount(g.MilestoneDeposit)
	timestamp := constraints.NewTimestamp(g.TimestampMilestone)
	chainConstraint := constraints.NewChainConstraint(g.MilestoneChainID, 0, 0, 0)
	stateCommitmentConstraint, err := constraints.NewGeneralScriptFromSource(fmt.Sprintf("id(0x%s)", genesisStateCommitment.String()))
	common.AssertNoError(err)

	ret := lazyslice.EmptyArray()
	ret.Push(amount.Bytes())
	ret.Push(timestamp.Bytes())
	ret.Push(g.MilestoneController.Bytes())
	ret.Push(chainConstraint.Bytes())
	ret.Push(stateCommitmentConstraint.Bytes())
	return ret.Bytes()
}

// NewReadable creates read-only ledger state with the given root
func NewReadable(store common.KVReader, root common.VCommitment) (*Readable, error) {
	trie, err := immutable.NewTrieReader(ledger.CommitmentModel, store, root)
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
