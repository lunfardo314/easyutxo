package constraint

import (
	"encoding/hex"
	"fmt"
)

func IsKnownLock(data []byte) bool {
	if _, ok := ParseAddressED25519Constraint(data); ok {
		return true
	}
	return false
}

func SigLockToString(lock []byte) string {
	if addr, ok := ParseAddressED25519Constraint(lock); ok {
		return fmt.Sprintf("addressED25519(0x%s)", hex.EncodeToString(addr))
	}
	return fmt.Sprintf("unknownConstraint(%s)", hex.EncodeToString(lock))
}

func UnlockParamsByReference(ref byte) []byte {
	return []byte{ref}
}
