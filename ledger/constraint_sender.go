package ledger

import (
	"github.com/lunfardo314/easyfl"
)

type Sender Constraint

func NewSenderConstraint(lock Lock, referencedInput byte) Constraint {
	return easyfl.Concat(byte(ConstraintTypeSender), referencedInput, lock)
}

func (s Sender) Lock() Lock {
	return Lock(s[2:])
}

const SenderConstraintSource = `

func selfReferencedInputIndex: byte(selfConstraintData, 0)
func selfReferencedLock: tail(selfConstraintData, 1)

func senderValid: or(
	isConsumedBranch(@),
	and(
		isProducedBranch(@),
		equal(selfReferencedLock, consumedLockByOutputIndex(selfReferencedInputIndex))
	)
)
`
