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
	sym, prefix, args, err := easyfl.ParseBinaryOneLevel(example, 3)
	easyfl.AssertNoError(err)
	easyfl.Assert(sym == "deadlineLock", "inconsistency in 'deadlineLock' 1")
	dlBin := easyfl.StripDataPrefix(args[0])
	easyfl.Assert(len(dlBin) == 4 && binary.BigEndian.Uint32(dlBin) == 1337, "inconsistency in 'deadlineLock' 2")
	easyfl.Assert(bytes.Equal(easyfl.StripDataPrefix(args[1]), easyfl.StripDataPrefix(AddressED25519LockNullBin())), "inconsistency in 'deadlineLock' 3")
	easyfl.Assert(bytes.Equal(easyfl.StripDataPrefix(args[2]), easyfl.StripDataPrefix(AddressED25519LockNullBin())), "inconsistency in 'deadlineLock' 4")

	registerConstraint("deadlineLock", prefix)
}

func ParseDeadlineLock(data []byte) (uint32, []byte, []byte, bool) {
	sym, _, args, err := easyfl.ParseBinaryOneLevel(data, 3)
	dlBin := easyfl.StripDataPrefix(args[0])
	if err != nil || sym != "deadlineLock" || len(dlBin) != 4 {
		return 0, nil, nil, false
	}
	return binary.BigEndian.Uint32(dlBin), args[1], args[2], true
}

func IsDeadlineLock(data []byte) bool {
	_, _, _, ok := ParseDeadlineLock(data)
	return ok
}

const deadlineLockSource = `

func deadlineLock: if(
	lessThan($0, txTimestampBytes),
	$1, 
	$2
)
`
