package library

import (
	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

func NewGlobalContext(dataTree *lazyslice.Tree, path lazyslice.TreePath) *GlobalContext {
	return &GlobalContext{
		dataTree:       dataTree,
		invocationPath: path,
	}
}

func (glb *GlobalContext) Eval(f *easyfl.Expression) []byte {
	return easyfl.NewRunContext(glb).Eval(f)
}

// EvalWithArgs pushes values for argument references
func (glb *GlobalContext) EvalWithArgs(f *easyfl.Expression, args ...[]byte) []byte {
	// TODO
	return glb.Eval(f)
}
