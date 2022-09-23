package library

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/easyfl"
)

var _ easyfl.LibraryAccess = &libraryData{}

func (lib *libraryData) ExistsFunction(sym string) bool {
	_, found := Library.funByName[sym]
	return found
}

func (lib *libraryData) FunctionByName(sym string) (*easyfl.FunInfo, error) {
	fd, found := Library.funByName[sym]
	if !found {
		return nil, fmt.Errorf("no such function in the library: '%s'", sym)
	}
	ret := &easyfl.FunInfo{
		Sym:       sym,
		FunCode:   fd.funCode,
		NumParams: fd.requiredNumParams,
	}
	switch {
	case fd.funCode < easyfl.FirstEmbeddedLongFun:
		ret.IsEmbedded = true
		ret.IsShort = true
	case fd.funCode < easyfl.FirstExtendedFun:
		ret.IsEmbedded = true
		ret.IsShort = false
	}
	return ret, nil
}

func (lib *libraryData) FunctionByCode(funCode uint16) (easyfl.EvalFunction, int, error) {
	var libData *funDescriptor
	libData = Library.funByFunCode[funCode]
	if libData == nil {
		return nil, 0, fmt.Errorf("wrong function code %d", funCode)
	}
	return libData.evalFun, libData.requiredNumParams, nil
}
