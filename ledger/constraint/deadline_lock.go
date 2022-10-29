package constraint

import (
	"bytes"
	"encoding/binary"
	"fmt"

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
	sym, prefix, args, err := easyfl.DecompileBinaryOneLevel(example, 3)
	easyfl.AssertNoError(err)
	easyfl.Assert(sym == "deadlineLock", "inconsistency in 'deadlineLock' 1")
	easyfl.Assert(len(args[0]) == 4 && binary.BigEndian.Uint32(args[0]) == 1337, "inconsistency in 'deadlineLock' 2")
	easyfl.Assert(bytes.Equal(args[1], AddressED25519LockNullBin()), "inconsistency in 'deadlineLock' 3")
	easyfl.Assert(bytes.Equal(args[2], AddressED25519LockNullBin()), "inconsistency in 'deadlineLock' 4")

	registerConstraint("deadlineLock", prefix)
}

func IsDeadlineLock(data []byte) bool {
	prefix, err := easyfl.ParseCallPrefixFromBinary(data)
	if err != nil {
		return false
	}
	prefix1, ok := PrefixByName("deadlineLock")
	if !ok {
		return false
	}
	return bytes.Equal(prefix, prefix1)
}

const deadlineLockSource = `

func deadlineLock: if(
	lessThan($0, txTimestampBytes),
	$1, 
	$2
)
`
