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

const FirstUserFunCode = 64 + 128

var (
	embeddedShortByName map[string]*funDef
	embeddedShortByCode [32]*funDef
	embeddedLongByName  map[string]*funDef
	embeddedLongByCode  map[uint16]*funDef
)

func init() {
	if len(embeddedShort) > 32 {
		panic("failed: len(embeddedShort) <= 32")
	}

	embeddedShortByName = make(map[string]*funDef)
	for i, fd := range embeddedShort {
		fd.funCode = uint16(i)
		if _, already := embeddedShortByName[fd.sym]; already {
			panic(fmt.Errorf("repeating symbol '%s'", fd.sym))
		}
		embeddedShortByName[fd.sym] = fd
	}

	embeddedLongByName = make(map[string]*funDef)
	for i, fd := range embeddedLong {
		fd.funCode = uint16(i) + MaxNumShortCall
		if _, already := embeddedLongByName[fd.sym]; already {
			panic(fmt.Errorf("repeating symbol '%s'", fd.sym))
		}
		embeddedLongByName[fd.sym] = fd
	}

	for _, fd := range embeddedShortByName {
		embeddedShortByCode[fd.funCode] = fd
	}

	embeddedLongByCode = make(map[uint16]*funDef)
	for _, fd := range embeddedLongByName {
		embeddedLongByCode[fd.funCode] = fd
	}
}
