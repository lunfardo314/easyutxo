package ledger

import (
	"fmt"
	"io"

	"github.com/lunfardo314/easyfl"
)

type (
	Sender struct {
		ReferencedInput byte
		Lock            Lock
	}
)

func (s *Sender) Bytes() []byte {
	return easyfl.Concat(ConstraintIDSender, s.ReferencedInput, s.Lock)
}

func SenderFromLock(lock Lock, referencedInput byte) *Sender {
	return &Sender{
		ReferencedInput: referencedInput,
		Lock:            lock,
	}
}

func SenderFromBytes(data []byte) (*Sender, error) {
	if len(data) < 3 {
		return nil, io.EOF
	}
	if data[0] != ConstraintIDSender {
		return nil, fmt.Errorf("expected sender constraint id %d", ConstraintIDSender)
	}
	return &Sender{
		ReferencedInput: data[1],
		Lock:            data[2:],
	}, nil
}

func SenderNull() *Sender {
	return &Sender{
		ReferencedInput: 0,
		Lock:            LockED25519Null(),
	}
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
