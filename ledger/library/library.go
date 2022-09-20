package library

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/lunfardo314/easyutxo/easyfl"
)

var embeddedShort = []*funDescriptor{
	// stateless
	{sym: "_slice", numParams: 3, evalFun: evalFun(evalSlice)},
	{sym: "_equal", numParams: 2, evalFun: evalFun(evalEqual)},
	{sym: "_len8", numParams: 1, evalFun: evalFun(evalLen8)},
	{sym: "_len16", numParams: 1, evalFun: evalFun(evalLen16)},
	{sym: "_not", numParams: 1, evalFun: evalFun(evalNot)},
	{sym: "_if", numParams: 3, evalFun: evalFun(evalIf)},
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
		mustValidAndUniqueName(fd.sym)
		fd.funCode = uint16(i)
		Library.embeddedShortByName[fd.sym] = fd
	}

	for i, fd := range embeddedLong {
		mustValidAndUniqueName(fd.sym)
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

func mustValidAndUniqueName(sym string) {
	if sym == "nil" || sym == "false" {
		panic("reserved symbol '" + sym + "'")
	}
	if Library.ExistsFunction(sym) {
		panic("repeating symbol '" + sym + "'")
	}
}

func evalFun(f runnerFunc) easyfl.EvalFunction {
	return func(glb interface{}, args []*easyfl.FormulaTree) []byte {
		return f(glb.(*RunContext), args)
	}
}

func evalSlice(glb *RunContext, args []*easyfl.FormulaTree) []byte {
	data := args[0].Eval(glb)
	from := args[1].Eval(glb)
	to := args[2].Eval(glb)
	return data[to[0]:from[0]]
}

func evalEqual(glb *RunContext, args []*easyfl.FormulaTree) []byte {
	p1 := args[0].Eval(glb)
	p2 := args[1].Eval(glb)
	if bytes.Equal(p1, p2) {
		return []byte{0xff}
	}
	return nil
}

func evalLen8(glb *RunContext, args []*easyfl.FormulaTree) []byte {
	data := args[0].Eval(glb)
	if len(data) > math.MaxUint8 {
		panic("len8: size of the data > 255")
	}
	return []byte{byte(len(data))}
}

func evalLen16(glb *RunContext, args []*easyfl.FormulaTree) []byte {
	data := args[0].Eval(glb)
	if len(data) > math.MaxUint16 {
		panic("len16: size of the data > uint16")
	}
	var ret [2]byte
	binary.BigEndian.PutUint16(ret[:], uint16(len(data)))
	return ret[:]
}

func evalIf(glb *RunContext, args []*easyfl.FormulaTree) []byte {
	cond := args[0].Eval(glb)
	if len(cond) != 0 {
		// true
		return args[1].Eval(glb)
	}
	return args[2].Eval(glb)
}

func evalNot(glb *RunContext, args []*easyfl.FormulaTree) []byte {
	if len(args[0].Eval(glb)) == 0 {
		return []byte{0xff}
	}
	return nil
}
