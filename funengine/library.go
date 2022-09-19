package funengine

import (
	"fmt"
)

var embeddedShort = []*funDef{
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
	// 9 left
}

var embeddedLong = []*funDef{
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
	embeddedShortByName    map[string]*funDef
	embeddedShortByFunCode [MaxNumShortCall]*funDef
	embeddedLongByName     map[string]*funDef
	embeddedLongByFunCode  map[uint16]*funDef
	extendedByName         map[string]*funDef
	extendedByFunCode      map[uint16]*funDef
}

var library = &libraryData{}

func init() {
	if len(embeddedShort) > MaxLongCallCode {
		panic("failed: len(embeddedShort) <= 32")
	}

	library.embeddedShortByName = make(map[string]*funDef)
	for i, fd := range embeddedShort {
		fd.funCode = uint16(i)
		if _, already := library.embeddedShortByName[fd.sym]; already {
			panic(fmt.Errorf("repeating symbol '%s'", fd.sym))
		}
		library.embeddedShortByName[fd.sym] = fd
	}

	library.embeddedLongByName = make(map[string]*funDef)
	for i, fd := range embeddedLong {
		fd.funCode = uint16(i) + MaxNumShortCall
		if _, already := library.embeddedLongByName[fd.sym]; already {
			panic(fmt.Errorf("repeating symbol '%s'", fd.sym))
		}
		library.embeddedLongByName[fd.sym] = fd
	}

	for _, fd := range library.embeddedShortByName {
		library.embeddedShortByFunCode[fd.funCode] = fd
	}

	library.embeddedLongByFunCode = make(map[uint16]*funDef)
	for _, fd := range library.embeddedLongByName {
		library.embeddedLongByFunCode[fd.funCode] = fd
	}
	library.extendedByName = make(map[string]*funDef)
	library.extendedByFunCode = make(map[uint16]*funDef)
}

func (lib *libraryData) Exists(sym string) bool {
	_, found := library.embeddedShortByName[sym]
	if found {
		return true
	}
	_, found = library.embeddedLongByName[sym]
	if found {
		return true
	}
	_, found = library.extendedByName[sym]
	if found {
		return true
	}
	return false
}

func (lib *libraryData) Resolve(sym string, numParams int) (*FunInfo, error) {
	if numParams > MaxParameters {
		return nil, fmt.Errorf("can't be more than 15 arguments in the call")
	}
	fd, found := library.embeddedShortByName[sym]
	if found {
		if fd.numParams != numParams {
			return nil, fmt.Errorf("'%s' require exactly %d arguments", sym, fd.numParams)
		}
		return &FunInfo{
			Sym:        sym,
			FunCode:    fd.funCode,
			IsEmbedded: true,
			IsShort:    true,
			NumParams:  fd.numParams,
		}, nil
	}
	fd, found = library.embeddedLongByName[sym]
	if found {
		if fd.numParams > 0 && fd.numParams != numParams {
			return nil, fmt.Errorf("'%s' require exactly %d arguments", sym, fd.numParams)
		}
		return &FunInfo{
			Sym:        sym,
			FunCode:    fd.funCode,
			IsEmbedded: true,
			IsShort:    false,
			NumParams:  fd.numParams,
		}, nil
	}
	fd, found = library.extendedByName[sym]
	if found {
		if fd.numParams < 0 {
			panic("internal error")
		}
		if fd.numParams != numParams {
			return nil, fmt.Errorf("'%s' require exactly %d arguments", sym, fd.numParams)
		}
		return &FunInfo{
			Sym:        sym,
			FunCode:    fd.funCode,
			IsEmbedded: false,
			IsShort:    false,
			NumParams:  fd.numParams,
		}, nil
	}
	return nil, fmt.Errorf("can't resolve function name '%s'", sym)
}

func (lib *libraryData) FunctionByFunCode(funCode uint16, arity int) func() {
	if funCode < MaxNumShortCall {
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
