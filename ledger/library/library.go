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
	return func(glb interface{}, args []*easyfl.FormulaTree) []byte {
		return f(glb.(*RunContext), args)
	}
}

func getArgFun(n byte) easyfl.EvalFunction {
	if n > 15 {
		panic("getArgFun: can be > 15")
	}
	return func(glb interface{}, _ []*easyfl.FormulaTree) []byte {
		return glb.(*RunContext).arg(n)
	}
}

func getCallFun(f runnerFunc) easyfl.EvalFunction {
	return func(glb interface{}, args []*easyfl.FormulaTree) []byte {
		argValues := make([][]byte, len(args))
		for i, a := range args{
			argValues[i] =  a.Eval(glb)
		}
		return glb.(*RunContext).Call(nil, argValues...)
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

func evalPath(glb *RunContext, _ []*easyfl.FormulaTree) []byte {
	return glb.invocationPath
}

func evalData(glb *RunContext, _ []*easyfl.FormulaTree) []byte {
	inv := glb.globalContext.BytesAtPath(glb.invocationPath)
	// TODO all kinds of invocation
	return inv[1:]
}

func evalAtPath(glb *RunContext, args []*easyfl.FormulaTree) []byte {
	return glb.globalContext.BytesAtPath(args[0].Eval(glb))
}

func evalConcat(glb *RunContext, args []*easyfl.FormulaTree) []byte {
	var buf bytes.Buffer
	for _, arg := range args {
		buf.Write(arg.Eval(glb))
	}
	return buf.Bytes()
}

func evalAnd(glb *RunContext, args []*easyfl.FormulaTree) []byte {
	for _, arg := range args {
		if len(arg.Eval(glb)) == 0 {
			return nil
		}
	}
	return []byte{0xff}
}

func evalOr(glb *RunContext, args []*easyfl.FormulaTree) []byte {
	for _, arg := range args {
		if len(arg.Eval(glb)) > 0 {
			return []byte{0xff}
		}
	}
	return nil
}

func evalBlake2b(glb *RunContext, args []*easyfl.FormulaTree) []byte {
	ret := blake2b.Sum256(evalConcat(glb, args))
	return ret[:]
}

func
