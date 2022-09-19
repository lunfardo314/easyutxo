package library

import (
	"fmt"

	"github.com/lunfardo314/easyutxo/easyfl"
)

type funEmbedded struct {
	sym       string
	funCode   uint16
	numParams int
	run       func()
}

type funExtended struct {
	sym        string
	funCode    uint16
	numParams  int
	binaryCode []byte
}

var embeddedShort = []*funEmbedded{
	{sym: "$0", numParams: 0},
	{sym: "$1", numParams: 0},
	{sym: "$2", numParams: 0},
	{sym: "$3", numParams: 0},
	{sym: "$4", numParams: 0},
	{sym: "$5", numParams: 0},
	{sym: "$6", numParams: 0},
	{sym: "$7", numParams: 0},
	{sym: "$8", numParams: 0},
	{sym: "$9", numParams: 0},
	{sym: "$10", numParams: 0},
	{sym: "$11", numParams: 0},
	{sym: "$12", numParams: 0},
	{sym: "$13", numParams: 0},
	{sym: "$14", numParams: 0},
	{sym: "$15", numParams: 0},
	{sym: "_data", numParams: 0},
	{sym: "_path", numParams: 0},
	{sym: "_slice", numParams: 3},
	{sym: "_atPath", numParams: 1},
	{sym: "_if", numParams: 3},
	{sym: "_equal", numParams: 2},
	{sym: "_len", numParams: 1},
	{sym: "_not", numParams: 1},
}

var embeddedLong = []*funEmbedded{
	{sym: "concat", numParams: -1},
	{sym: "and", numParams: -1},
	{sym: "or", numParams: -1},
	{sym: "blake2b", numParams: -1},
	{sym: "validSignature", numParams: 3},
}

const (
	MaxParameters    = 15
	FirstUserFunCode = 64 + 128
)

type libraryData struct {
	embeddedShortByName    map[string]*funEmbedded
	embeddedShortByFunCode [easyfl.MaxNumShortCall]*funEmbedded
	embeddedLongByName     map[string]*funEmbedded
	embeddedLongByFunCode  map[uint16]*funEmbedded
	extendedByName         map[string]*funExtended
	extendedByFunCode      map[uint16]*funExtended
}

var Library = &libraryData{}

func init() {
	if len(embeddedShort) > easyfl.MaxLongCallCode {
		panic("failed: len(embeddedShort) <= MaxLongCallCode")
	}

	Library.embeddedShortByName = make(map[string]*funEmbedded)
	for i, fd := range embeddedShort {
		fd.funCode = uint16(i)
		if _, already := Library.embeddedShortByName[fd.sym]; already {
			panic(fmt.Errorf("repeating symbol '%s'", fd.sym))
		}
		Library.embeddedShortByName[fd.sym] = fd
	}

	Library.embeddedLongByName = make(map[string]*funEmbedded)
	for i, fd := range embeddedLong {
		fd.funCode = uint16(i) + easyfl.MaxNumShortCall
		if _, already := Library.embeddedLongByName[fd.sym]; already {
			panic(fmt.Errorf("repeating symbol '%s'", fd.sym))
		}
		Library.embeddedLongByName[fd.sym] = fd
	}

	for _, fd := range Library.embeddedShortByName {
		Library.embeddedShortByFunCode[fd.funCode] = fd
	}

	Library.embeddedLongByFunCode = make(map[uint16]*funEmbedded)
	for _, fd := range Library.embeddedLongByName {
		Library.embeddedLongByFunCode[fd.funCode] = fd
	}
	Library.extendedByName = make(map[string]*funExtended)
	Library.extendedByFunCode = make(map[uint16]*funExtended)
}

func (lib *libraryData) ExistsFun(sym string) bool {
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

func (lib *libraryData) Resolve(sym string, numParams int) (*easyfl.FunInfo, error) {
	if numParams > MaxParameters {
		return nil, fmt.Errorf("can't be more than 15 arguments in the call")
	}
	fd, found := Library.embeddedShortByName[sym]
	if found {
		if fd.numParams != numParams {
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
		if fd.numParams > 0 && fd.numParams != numParams {
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
		if fe.numParams != numParams {
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

func (lib *libraryData) FunctionByFunCode(funCode uint16, arity int) func() {
	if funCode < easyfl.MaxNumShortCall {
		ret := lib.embeddedShortByFunCode[funCode]
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
	ret, found := lib.embeddedLongByFunCode[funCode]
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
	_, found = lib.extendedByFunCode[funCode]
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

func (lib *libraryData) addToLibrary(fe *funExtended) error {
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
		code, err := easyfl.CompileFormula(lib, pf.NumParams, pf.SourceCode)
		if err != nil {
			return err
		}
		err = lib.addToLibrary(&funExtended{
			sym:        pf.Sym,
			numParams:  pf.NumParams,
			binaryCode: code,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
