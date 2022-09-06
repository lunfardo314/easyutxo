package opcodes

import (
	"github.com/lunfardo314/easyutxo/engine"
	"golang.org/x/crypto/ed25519"
)

// runSigLockED25519 expects at the stack top
// - essence bytes -> removed
// - signature -> removed
// - public key -> not removed
func runSigLockED25519(e *engine.Engine, d []byte) {
	mustParLen(d, 0)
	essence := e.Pop()
	signature := e.Pop()
	pubKey := ed25519.PublicKey(e.Top())
	e.PushBool(ed25519.Verify(pubKey, essence, signature))
	e.Move(1)
}
