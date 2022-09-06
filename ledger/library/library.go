package library

import (
	"github.com/lunfardo314/easyutxo/lazyslice"
)

var ScriptLibrary *lazyslice.Tree

var scriptNA = []byte{0xFF}

func init() {
	ScriptLibrary = lazyslice.TreeEmpty()

	for i := 0; i < 256; i++ {
		ScriptLibrary.PushData(scriptNA, nil)
	}

	ScriptLibrary.PutDataAtIdx(LibScriptSigLockED25519, SigLockED25519, nil)
}

const (
	CodeReservedForLocalInvocations = byte(iota)
	CodeReservedForInlineInvocations

	// indices of library scripts

	LibScriptSigLockED25519 = byte(iota)
)
