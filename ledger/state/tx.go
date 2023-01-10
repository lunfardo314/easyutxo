package state

import (
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger"
	"github.com/lunfardo314/easyutxo/ledger/constraints"
	"github.com/lunfardo314/unitrie/common"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/ed25519"
)

// Transaction provides access to the tree of transferable transaction
type Transaction struct {
	tree   *lazyslice.Tree
	txid   ledger.TransactionID
	sender constraints.AddressED25519
}

func MustTransactionFromTransferableBytes(txBytes []byte) (*Transaction, error) {
	ret := &Transaction{
		tree: lazyslice.TreeFromBytes(txBytes),
		txid: blake2b.Sum256(txBytes),
	}
	senderPubKey := ed25519.PublicKey(ret.tree.BytesAtPath(Path(constraints.TxSignature)))
	ret.sender = constraints.AddressED25519FromPublicKey(senderPubKey)
	if err := ret.mustPreValidate(); err != nil {
		return nil, err
	}
	return ret, nil
}

// mustPreValidate validates what is possible without ledger context
func (tx *Transaction) mustPreValidate() error {
	if err := tx.checkNumElements(); err != nil {
		return err
	}
	if err := tx.checkUniqueness(); err != nil {
		return err
	}
	if err := tx.checkMandatory(); err != nil {
		return err
	}
	if err := tx.checkOther(); err != nil {
		return err
	}
	return nil
}

func (tx *Transaction) checkNumElements() error {
	if tx.tree.NumElements(Path(constraints.TxOutputs)) <= 0 {
		return fmt.Errorf("number of outputs can't be 0")
	}

	numInputs := tx.tree.NumElements(Path(constraints.TxInputIDs))
	if numInputs <= 0 {
		return fmt.Errorf("number of inputs can't be 0")
	}

	if numInputs != tx.tree.NumElements(Path(constraints.TxUnlockParams)) {
		return fmt.Errorf("number of unlock params must be equal to the number of inputs")
	}

	if tx.tree.NumElements(Path(constraints.TxEndorsements)) > constraints.MaxNumberOfEndorsements {
		return fmt.Errorf("number of endorsements exceeds limit of %d", constraints.MaxNumberOfEndorsements)
	}
	return nil
}

func (tx *Transaction) checkUniqueness() error {
	var err error
	// check if inputs are unique
	inps := make(map[ledger.OutputID]struct{})
	tx.MustForEachInput(func(i byte, oid *ledger.OutputID) bool {
		_, already := inps[*oid]
		if already {
			err = fmt.Errorf("repeating input @ %d", i)
			return false
		}
		inps[*oid] = struct{}{}
		return true
	})
	if err != nil {
		return err
	}

	// check if endorsements are unique
	endorsements := make(map[ledger.TransactionID]struct{})
	tx.MustForEachEndorsement(func(i byte, txid ledger.TransactionID) bool {
		_, already := endorsements[txid]
		if already {
			err = fmt.Errorf("repeating endorsement @ %d", i)
			return false
		}
		endorsements[txid] = struct{}{}
		return true
	})
	if err != nil {
		return err
	}

	return nil
}

func (tx *Transaction) checkMandatory() error {
	numOutputs := tx.tree.NumElements(Path(constraints.TxOutputs))
	for i := 0; i < numOutputs; i++ {
		bytecode := tx.tree.BytesAtPath(Path(constraints.TxOutputs, constraints.ConstraintIndexAmount))
		fname, _, _, err := easyfl.ParseBytecodeOneLevel(bytecode, 1)
		if err != nil {
			return err
		}
		if fname != constraints.AmountConstraintName {
			return fmt.Errorf("'%s' constraint expected at position %d in the output #%d",
				constraints.AmountConstraintName, constraints.ConstraintIndexAmount, i)
		}

		bytecode = tx.tree.BytesAtPath(Path(constraints.TxOutputs, constraints.ConstraintIndexTimestamp))
		fname, _, _, err = easyfl.ParseBytecodeOneLevel(bytecode, 1)
		if err != nil {
			return err
		}
		if fname != constraints.TimestampConstraintName {
			return fmt.Errorf("'%s' constraint expected at position %d in the output #%d",
				constraints.TimestampConstraintName, constraints.ConstraintIndexAmount, i)
		}

		bytecode = tx.tree.BytesAtPath(Path(constraints.TxOutputs, constraints.ConstraintIndexLock))
		fname, _, _, err = easyfl.ParseBytecodeOneLevel(bytecode, 1)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tx *Transaction) checkOther() error {
	data := tx.tree.BytesAtPath(Path(constraints.TxSignature))
	if !ed25519.Verify(data[64:], tx.tree.Bytes(), data[0:64]) {
		return fmt.Errorf("invalid signature")
	}
	data = tx.tree.BytesAtPath(Path(constraints.TxInputCommitment))
	if len(data) != 32 {
		return fmt.Errorf("input commitment must be 32-bytes long")
	}
	return nil
}

func (tx *Transaction) ID() ledger.TransactionID {
	return tx.txid
}

func (tx *Transaction) SenderAddress() constraints.AddressED25519 {
	return tx.sender
}

func (tx *Transaction) NumProducedOutputs() int {
	return tx.tree.NumElements(Path(constraints.TxOutputs))
}

func (tx *Transaction) NumInputs() int {
	return tx.tree.NumElements(Path(constraints.TxInputIDs))
}

func (tx *Transaction) OutputAt(idx byte) []byte {
	return tx.tree.BytesAtPath(common.Concat(constraints.TxOutputs, idx))
}

func (tx *Transaction) MustForEachInput(fun func(i byte, oid *ledger.OutputID) bool) {
	tx.tree.ForEach(func(i byte, data []byte) bool {
		oid, err := ledger.OutputIDFromBytes(data)
		common.Assert(err == nil, "MustForEachInput @ %d: %v", i, err)
		return fun(i, &oid)
	}, Path(constraints.TxInputIDs))
}

func (tx *Transaction) MustForEachEndorsement(fun func(idx byte, txid ledger.TransactionID) bool) {
	tx.tree.ForEach(func(i byte, data []byte) bool {
		txid, err := ledger.TransactionIDFromBytes(data)
		common.Assert(err == nil, "MustForEachEndorsement @ %d: %v", i, err)
		return fun(i, txid)
	}, Path(constraints.TxEndorsements))
}

// MustForEachConsumedTransactionID iterates over unique transaction IDs consumed in
// the transaction in the order of appearance
func (tx *Transaction) MustForEachConsumedTransactionID(fun func(txid *ledger.TransactionID)) {
	already := make(map[ledger.TransactionID]struct{})
	tx.MustForEachInput(func(i byte, oid *ledger.OutputID) bool {
		txid := oid.TransactionID()
		if _, found := already[txid]; !found {
			already[txid] = struct{}{}
			fun(&txid)
		}
		return true
	})
}
