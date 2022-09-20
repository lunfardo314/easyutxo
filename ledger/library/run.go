package library

import (
	"github.com/lunfardo314/easyutxo/lazyslice"
)

func NewRunContext(glb *lazyslice.Tree, path lazyslice.TreePath) *RunContext {
	return &RunContext{
		globalContext:  glb,
		invocationPath: path,
	}
}
