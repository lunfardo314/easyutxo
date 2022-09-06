package library

import (
	"testing"
)

func TestScripts(t *testing.T) {
	t.Run("SigLockED25519", func(t *testing.T) {
		t.Logf("SigLockED25519 len = %d", len(SigLockED25519))
		t.Logf("SigLockED25519: %v", SigLockED25519)
	})
}
