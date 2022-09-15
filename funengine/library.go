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

const FirstUserFunCode = 1024

var (
	embeddedShortByName = mustPreCompileEmbeddedShortBySym()
	embeddedLongByName  = mustPreCompileEmbeddedLongBySym()
)

func mustMakeMapBySym(defs []*funDef) map[string]*funDef {
	ret := make(map[string]*funDef)
	for i, fd := range defs {
		fd.funCode = uint16(i)
		if _, already := ret[fd.sym]; already {
			panic(fmt.Errorf("repeating symbol '%s'", fd.sym))
		}
		ret[fd.sym] = fd
	}
	return ret
}

func mustPreCompileEmbeddedShortBySym() map[string]*funDef {
	if len(embeddedShort) > 32 {
		panic("failed: len(embeddedShort) <= 32")
	}
	ret := mustMakeMapBySym(embeddedShort)
	for _, fd := range ret {
		if fd.numParams < 0 {
			panic(fmt.Errorf("embedded short must be fixed number of parameters: '%s'", fd.sym))
		}
	}
	return ret
}

func mustPreCompileEmbeddedLongBySym() map[string]*funDef {
	ret := mustMakeMapBySym(embeddedLong)
	// offset fun codes by 32
	for _, fd := range ret {
		fd.funCode += 32
	}
	return ret
}
