package library

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
)

func NewRunContext(glb *lazyslice.Tree, path lazyslice.TreePath) *RunContext {
	return &RunContext{
		globalContext:  glb,
		invocationPath: path,
		stack:          make([][]byte, maxStack),
		stackTop:       0,
	}
}

func (ctx *RunContext) FunctionByFunCode(funCode uint16, arity int) func() {
	if funCode < easyfl.MaxNumShortCall {
		ret := Library.embeddedShortByFunCode[funCode]
		if ret == nil {
			return func() {
				panic("reserved short code")
			}
		}
		if arity != ret.numParams {
			return func() {
				// TODO temporary
				fmt.Printf("dummy run short call %d with WRONG call arity %d\n", funCode, arity)
			}
		}
		return func() {
			// TODO temporary
			fmt.Printf("dummy run short call %d with call arity %d\n", funCode, arity)
		}
	}
	ret, found := Library.embeddedLongByFunCode[funCode]
	if found {
		if ret.numParams >= 0 && arity != ret.numParams {
			return func() {
				// TODO temporary
				fmt.Printf("dummy run long call %d with WRONG call arity %d\n", funCode, arity)
			}
		}
		return func() {
			// TODO temporary
			fmt.Printf("dummy run long call %d with call arity %d\n", funCode, arity)
		}
	}
	_, found = Library.extendedByFunCode[funCode]
	if found {
		if arity != ret.numParams {
			return func() {
				// TODO temporary
				fmt.Printf("dummy run extended call %d with WRONG call arity %d\n", funCode, arity)
			}
		}
		return func() {
			// TODO temporary
			fmt.Printf("dummy run extended call %d\n", funCode)
		}
	}
	return func() {
		panic(fmt.Errorf("funCode %d not found", funCode))
	}
}
