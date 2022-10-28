package constraint

import (
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
)

func DeadlineLockSource(deadline uint32, mainOptionSource, expirySource string) string {
	return fmt.Sprintf("deadlineLock(u32/%d,%s,%s)", deadline, mainOptionSource, expirySource)
}

func DeadlineLock(deadline uint32, main, expiry string) []byte {
	return mustBinFromSource(DeadlineLockSource(deadline, main, expiry))
}

func initDeadlineLockConstraint() {
	easyfl.MustExtendMany(deadlineLockSource)

	example := DeadlineLock(1337, AddressED25519LockNullSource(), AddressED25519LockNullSource())
	prefix, err := easyfl.ParseCallPrefixFromBinary(example)
	common.AssertNoError(err)
	registerConstraint("deadlineLock", prefix)
}

const deadlineLockSource = `

func deadlineLock: if(
	lessThan($0, txTimestampBytes),
	$1, 
	$2
)
`
