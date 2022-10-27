package ledger

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
)

// TODO take sender from unlock block or only accept senders from known types of locks

func SenderConstraint(lock Constraint, referencedInput byte) Constraint {
	src := fmt.Sprintf("sender(0x%s, %d)", hex.EncodeToString(lock), referencedInput)
	_, _, binCode, err := easyfl.CompileExpression(src)
	easyfl.AssertNoError(err)
	return binCode
}

var (
	senderConstraintPrefix []byte
)

func initSenderConstraint() {
	prefix, err := easyfl.FunctionCodeBytesByName("sender")
	easyfl.AssertNoError(err)
	common.Assert(0 < len(prefix) && len(prefix) <= 2, "0<len(prefix) && len(prefix)<=2")
	template := SenderConstraint(AddressED25519SigLockNull(), 0)
	lenConstraintPrefix := len(prefix) + 1
	common.Assert(len(template) == len(prefix)+1+8, "len(template)==len(prefix)+1+8")
	senderConstraintPrefix = easyfl.Concat(template[:lenConstraintPrefix])
}

// SenderFromConstraint extracts sender address ($0) from the sender script
func SenderFromConstraint(data []byte) ([]byte, error) {
	if !bytes.HasPrefix(data, senderConstraintPrefix) {
		return nil, fmt.Errorf("SenderFromConstraint:: not a sender constraint")
	}
	if len(data) < len(senderConstraintPrefix)+1 {
		return nil, fmt.Errorf("SenderFromConstraint:: wrong data len")
	}
	senderAddr, success, err := easyfl.ParseInlineDataPrefix(data[len(senderConstraintPrefix):])
	if err != nil || !success {
		return nil, fmt.Errorf("failed to parse sender address")
	}
	return senderAddr, nil
}

const SenderConstraintSource = `

// $0 - lock constraint which is sender
// $1 - referenced consumed output index
func sender: or(
	isConsumedBranch(@),
	and(
		isProducedBranch(@),
		equal($0, consumedLockByOutputIndex($1))
	)
)
`
