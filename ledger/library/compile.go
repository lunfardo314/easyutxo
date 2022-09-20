package library

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/easyfl"
)

var _ easyfl.LibraryAccess = &libraryData{}

func (lib *libraryData) ExistsFunction(sym string) bool {
	_, found := Library.embeddedShortByName[sym]
	if found {
		return true
	}
	_, found = Library.embeddedLongByName[sym]
	if found {
		return true
	}
	_, found = Library.extendedByName[sym]
	if found {
		return true
	}
	return false
}

func (lib *libraryData) FunctionByName(sym string, numArgs int) (*easyfl.FunInfo, error) {
	if numArgs > easyfl.MaxParameters {
		return nil, fmt.Errorf("can't be more than 15 arguments in the call")
	}
	fd, found := Library.embeddedShortByName[sym]
	if found {
		if fd.numParams != numArgs {
			return nil, fmt.Errorf("'%s' require exactly %d arguments", sym, fd.numParams)
		}
		return &easyfl.FunInfo{
			Sym:        sym,
			FunCode:    fd.funCode,
			IsEmbedded: true,
			IsShort:    true,
			NumParams:  fd.numParams,
		}, nil
	}
	fd, found = Library.embeddedLongByName[sym]
	if found {
		if fd.numParams > 0 && fd.numParams != numArgs {
			return nil, fmt.Errorf("'%s' require exactly %d arguments", sym, fd.numParams)
		}
		return &easyfl.FunInfo{
			Sym:        sym,
			FunCode:    fd.funCode,
			IsEmbedded: true,
			IsShort:    false,
			NumParams:  fd.numParams,
		}, nil
	}
	fe, found := Library.extendedByName[sym]
	if found {
		if fe.numParams < 0 {
			panic("internal error")
		}
		if fe.numParams != numArgs {
			return nil, fmt.Errorf("'%s' require exactly %d arguments", sym, fe.numParams)
		}
		return &easyfl.FunInfo{
			Sym:        sym,
			FunCode:    fe.funCode,
			IsEmbedded: false,
			IsShort:    false,
			NumParams:  fe.numParams,
		}, nil
	}
	return nil, fmt.Errorf("can't resolve function name '%s'", sym)
}

func (lib *libraryData) FunctionByCode(funCode uint16) (easyfl.EvalFunction, int, error) {
	var libData *funDescriptor
	if funCode < easyfl.MaxNumShortCall {
		libData = Library.embeddedShortByFunCode[funCode]
	}
	if libData == nil {
		libData = Library.embeddedLongByFunCode[funCode]
	}
	if libData == nil {
		libData = Library.extendedByFunCode[funCode]
	}
	if libData != nil {
		return libData.evalFun, libData.numParams, nil
	}
	return nil, 0, fmt.Errorf("wrong function code %d", funCode)
}

func (lib *libraryData) addToLibrary(fe *funDescriptor) error {
	if lib.ExistsFun(fe.sym) {
		return fmt.Errorf("repeating function name '%s'", fe.sym)
	}
	fe.funCode = uint16(len(lib.extendedByName) + easyfl.ExtendedCodeOffset)
	lib.extendedByName[fe.sym] = fe
	lib.extendedByFunCode[fe.funCode] = fe
	return nil
}

func (lib *libraryData) compileAndAddMany(parsed []*easyfl.FunParsed) error {
	for _, pf := range parsed {
		_, err := easyfl.FormulaSourceToBinary(lib, pf.NumParams, pf.SourceCode)
		if err != nil {
			return err
		}
		err = lib.addToLibrary(&funDescriptor{
			sym:       pf.Sym,
			numParams: pf.NumParams,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
