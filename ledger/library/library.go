package library

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/lunfardo314/easyutxo/easyfl"
	"golang.org/x/crypto/blake2b"
)

var embeddedShort = []*funDescriptor{
	// stateless
	{sym: "_slice", requiredNumParams: 3, evalFun: getEvalFun(evalSlice)},
	{sym: "_equal", requiredNumParams: 2, evalFun: getEvalFun(evalEqual)},
	{sym: "_len8", requiredNumParams: 1, evalFun: getEvalFun(evalLen8)},
	{sym: "_len16", requiredNumParams: 1, evalFun: getEvalFun(evalLen16)},
	{sym: "_not", requiredNumParams: 1, evalFun: getEvalFun(evalNot)},
	{sym: "_if", requiredNumParams: 3, evalFun: getEvalFun(evalIf)},
	{sym: "_isZero", requiredNumParams: 1, evalFun: getEvalFun(evalIsZero)},
	{sym: "_sum8_16", requiredNumParams: 2, evalFun: getEvalFun(evalSum8_16)},
	// argument access
	{sym: "$0", requiredNumParams: 0, evalFun: getArgFun(0)},
	{sym: "$1", requiredNumParams: 0, evalFun: getArgFun(1)},
	{sym: "$2", requiredNumParams: 0, evalFun: getArgFun(2)},
	{sym: "$3", requiredNumParams: 0, evalFun: getArgFun(3)},
	{sym: "$4", requiredNumParams: 0, evalFun: getArgFun(4)},
	{sym: "$5", requiredNumParams: 0, evalFun: getArgFun(5)},
	{sym: "$6", requiredNumParams: 0, evalFun: getArgFun(6)},
	{sym: "$7", requiredNumParams: 0, evalFun: getArgFun(7)},
	{sym: "$8", requiredNumParams: 0, evalFun: getArgFun(8)},
	{sym: "$9", requiredNumParams: 0, evalFun: getArgFun(9)},
	{sym: "$10", requiredNumParams: 0, evalFun: getArgFun(10)},
	{sym: "$11", requiredNumParams: 0, evalFun: getArgFun(11)},
	{sym: "$12", requiredNumParams: 0, evalFun: getArgFun(12)},
	{sym: "$13", requiredNumParams: 0, evalFun: getArgFun(13)},
	{sym: "$14", requiredNumParams: 0, evalFun: getArgFun(14)},
	{sym: "$15", requiredNumParams: 0, evalFun: getArgFun(15)},
	// context access
	{sym: "_data", requiredNumParams: 0, evalFun: getEvalFun(evalData)},
	{sym: "_path", requiredNumParams: 0, evalFun: getEvalFun(evalPath)},
	{sym: "_atPath", requiredNumParams: 1, evalFun: getEvalFun(evalAtPath)},
}

var embeddedLong = []*funDescriptor{
	// stateless varargs
	{sym: "concat", requiredNumParams: -1, evalFun: getEvalFun(evalConcat)},
	{sym: "and", requiredNumParams: -1, evalFun: getEvalFun(evalAnd)},
	{sym: "or", requiredNumParams: -1, evalFun: getEvalFun(evalOr)},
	{sym: "blake2b", requiredNumParams: -1, evalFun: getEvalFun(evalBlake2b)},
	// special
	{sym: "validSignature", requiredNumParams: 3},
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

func getEvalFun(f runnerFunc) easyfl.EvalFunction {
	return func(glb easyfl.EvalContext) []byte {
		return f(glb.(*RunContext))
	}
}

func getArgFun(n byte) easyfl.EvalFunction {
	if n > 15 {
		panic("getArgFun: can be > 15")
	}
	return func(glb easyfl.EvalContext) []byte {
		return glb.(*RunContext).arg(n)
	}
}

func evalSlice(glb *RunContext) []byte {
	data := glb.arg(0)
	from := glb.arg(1)
	to := glb.arg(2)
	return data[from[0]:to[0]]
}

func evalEqual(glb *RunContext) []byte {
	if bytes.Equal(glb.arg(0), glb.arg(1)) {
		return []byte{0xff}
	}
	return nil
}

func evalLen8(glb *RunContext) []byte {
	sz := len(glb.arg(0))
	if sz > math.MaxUint8 {
		panic("len8: size of the data > 255")
	}
	return []byte{byte(sz)}
}

func evalLen16(glb *RunContext) []byte {
	data := glb.arg(0)
	if len(data) > math.MaxUint16 {
		panic("len16: size of the data > uint16")
	}
	var ret [2]byte
	binary.BigEndian.PutUint16(ret[:], uint16(len(data)))
	return ret[:]
}

func evalIf(glb *RunContext) []byte {
	cond := glb.arg(0)
	if len(cond) != 0 {
		// true
		return glb.arg(1)
	}
	return glb.arg(2)
}

func evalIsZero(glb *RunContext) []byte {
	for _, b := range glb.arg(0) {
		if b != 0 {
			return nil
		}
	}
	return []byte{0xff}
}

func evalSum8_16(glb *RunContext) []byte {
	sum := uint16(glb.arg(0)[0]) + uint16(glb.arg(1)[0])
	ret := make([]byte, 2)
	binary.BigEndian.PutUint16(ret, sum)
	return ret
}

func evalNot(glb *RunContext) []byte {
	if len(glb.arg(0)) == 0 {
		return []byte{0xff}
	}
	return nil
}

func evalPath(glb *RunContext) []byte {
	return glb.invocationPath
}

func evalData(glb *RunContext) []byte {
	inv := glb.globalContext.BytesAtPath(glb.invocationPath)
	// TODO all kinds of invocation
	return inv[1:]
}

func evalAtPath(glb *RunContext) []byte {
	return glb.globalContext.BytesAtPath(glb.arg(0))
}

func evalConcat(glb *RunContext) []byte {
	var buf bytes.Buffer
	for i := byte(0); i < glb.arity(); i++ {
		buf.Write(glb.arg(i))
	}
	return buf.Bytes()
}

func evalAnd(glb *RunContext) []byte {
	for i := byte(0); i < glb.arity(); i++ {
		if len(glb.arg(i)) == 0 {
			return nil
		}
	}
	return []byte{0xff}
}

func evalOr(glb *RunContext) []byte {
	for i := byte(0); i < glb.arity(); i++ {
		if len(glb.arg(i)) != 0 {
			return []byte{0xff}
		}
	}
	return nil
}

func evalBlake2b(glb *RunContext) []byte {
	ret := blake2b.Sum256(evalConcat(glb))
	return ret[:]
}
