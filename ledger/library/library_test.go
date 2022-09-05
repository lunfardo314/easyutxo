package library

import (
	"testing"
)

func TestScripts(t *testing.T) {
	t.Run("AddressED25519SigLock", func(t *testing.T) {
		t.Logf("AddressED25519SigLock len = %d", len(AddressED25519SigLock))
		t.Logf("AddressED25519SigLock: %v", AddressED25519SigLock)
	})
}
