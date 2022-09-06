package opcodes

import (
	"github.com/lunfardo314/easyutxo/engine"
	"golang.org/x/crypto/blake2b"
)

func runBlake2b(e *engine.Engine, d []byte) {
	mustParLen(d, 0)
	r := blake2b.Sum256(e.Top())
	e.Pop()
	e.Push(r[:])
	e.Move(1)
}
