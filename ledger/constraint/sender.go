package constraint

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
)

// TODO take sender from unlock block or only accept senders from known types of locks

func SenderConstraintSource(lock []byte, referencedInput byte) string {
	return fmt.Sprintf("sender(0x%s, %d)", hex.EncodeToString(lock), referencedInput)
}

func SenderConstraintBin(lock []byte, referencedInput byte) []byte {
	return mustBinFromSource(SenderConstraintSource(lock, referencedInput))
}

func initSenderConstraint() {
	easyfl.MustExtendMany(senderSource)

	addr := AddressED25519LockNullBin()
	example := SenderConstraintBin(addr, 0)
	sym, prefix, args, err := easyfl.ParseBinaryOneLevel(example, 2)
	easyfl.AssertNoError(err)
	common.Assert(sym == "sender" && bytes.Equal(args[0], addr), "inconsistency in 'sender'")
	registerConstraint("sender", prefix)
}

// SenderFromConstraint extracts sender address ($0) from the sender script
func SenderFromConstraint(data []byte) ([]byte, bool) {
	sym, _, args, err := easyfl.ParseBinaryOneLevel(data, 2)
	if err != nil {
		return nil, false
	}
	if sym != "sender" {
		return nil, false
	}
	return args[0], true
}

const senderSource = `

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
