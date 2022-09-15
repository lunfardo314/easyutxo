package funengine

import (
	"github.com/lunfardo314/easyutxo/lazyslice"
)

type RunContext struct {
	localLibrary map[uint16]*funDef
	//globalContext ledger.GlobalContext
}

type InvocationContext struct {
	runContext *RunContext
	path       lazyslice.TreePath
	data       []byte
	callStack  interface{}
}
