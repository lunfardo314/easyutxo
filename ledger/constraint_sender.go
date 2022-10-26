package ledger

import (
	"encoding/hex"
	"fmt"

	"github.com/lunfardo314/easyfl"
)

// TODO take sender from unlock block or only accept senders from known types of locks

func SenderConstraint(lock Constraint, referencedInput byte) Constraint {
	src := fmt.Sprintf("sender(0x%s, %d)", hex.EncodeToString(lock), referencedInput)
	_, _, binCode, err := easyfl.CompileExpression(src)
	easyfl.AssertNoError(err)
	return binCode
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
