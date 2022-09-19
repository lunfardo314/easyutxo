package library

import (
	"bytes"

	"github.com/lunfardo314/easyutxo/easyfl"
)

var embeddedShort = []*funDescriptor{
	// stateless
	{sym: "_slice", numParams: 3, getRunner: statelessRunner(runSlice)},
	{sym: "_equal", numParams: 2, getRunner: statelessRunner(runEqual)},
	{sym: "_len", numParams: 1},
	{sym: "_not", numParams: 1},
	{sym: "_if", numParams: 3},
	// argument access
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
	// context access
	{sym: "_data", numParams: 0},
	{sym: "_path", numParams: 0},
	{sym: "_atPath", numParams: 1},
}

var embeddedLong = []*funDescriptor{
	// stateless varargs
	{sym: "concat", numParams: -1},
	{sym: "and", numParams: -1},
	{sym: "or", numParams: -1},
	{sym: "blake2b", numParams: -1},
	// special
	{sym: "validSignature", numParams: 3},
}

type libraryData struct {
	embeddedShortByName    map[string]*funDescriptor
	embeddedShortByFunCode [easyfl.MaxNumShortCall]*funDescriptor
	embeddedLongByName     map[string]*funDescriptor
	embeddedLongByFunCode  map[uint16]*funDescriptor
	extendedByName         map[string]*funDescriptor
	extendedByFunCode      map[uint16]*funDescriptor
}

var Library = &libraryData{}

func init() {
	if len(embeddedShort) > easyfl.MaxNumShortCall {
		panic("failed: len(embeddedShort) <= MaxLongCallCode")
	}
	Library = &libraryData{
		embeddedShortByName:    make(map[string]*funDescriptor),
		embeddedShortByFunCode: [easyfl.MaxNumShortCall]*funDescriptor{},
		embeddedLongByName:     make(map[string]*funDescriptor),
		embeddedLongByFunCode:  make(map[uint16]*funDescriptor),
		extendedByName:         make(map[string]*funDescriptor),
		extendedByFunCode:      make(map[uint16]*funDescriptor),
	}
	for i, fd := range embeddedShort {
		mustUniqueName(fd.sym)
		fd.funCode = uint16(i)
		Library.embeddedShortByName[fd.sym] = fd
	}

	for i, fd := range embeddedLong {
		mustUniqueName(fd.sym)
		fd.funCode = uint16(i) + easyfl.MaxNumShortCall
		Library.embeddedLongByName[fd.sym] = fd
	}

	for _, fd := range Library.embeddedShortByName {
		Library.embeddedShortByFunCode[fd.funCode] = fd
	}

	for _, fd := range Library.embeddedLongByName {
		Library.embeddedLongByFunCode[fd.funCode] = fd
	}
}

func mustUniqueName(sym string) {
	if Library.ExistsFun(sym) {
		panic("repeating symbol '" + sym + "'")
	}
}

func statelessRunner(f runnerFunc) getRunnerFunc {
	return func(_ byte) runnerFunc {
		return f
	}
}

func runSlice(ctx *RunContext) []byte {
	to := ctx.Pop()
	from := ctx.Pop()
	data := ctx.Pop()
	return data[to[0]:from[0]]
}

func runEqual(ctx *RunContext) []byte {
	p1 := ctx.Pop()
	p2 := ctx.Pop()
	if bytes.Equal(p1, p2) {
		return []byte{0xff}
	}
	return nil
}
