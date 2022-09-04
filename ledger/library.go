package ledger

import "github.com/lunfardo314/easyutxo/lazyslice"

var ScriptLibrary *lazyslice.Tree

var scriptNA = []byte{0xFF}

func init() {
	ScriptLibrary = lazyslice.TreeEmpty()

	for i := 0; i < 256; i++ {
		ScriptLibrary.PushData(scriptNA, nil)
	}
}

const (
	LibraryCodeReservedForLocalInvocations = byte(iota)
	LibraryCodeReservedForInlineInvocations
)
