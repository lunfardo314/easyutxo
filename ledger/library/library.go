package library

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/lunfardo314/easyutxo/easyfl"
	"golang.org/x/crypto/blake2b"
)

type libraryData struct {
	funByName    map[string]*funDescriptor
	funByFunCode map[uint16]*funDescriptor
}

var (
	Library = &libraryData{
		funByName:    make(map[string]*funDescriptor),
		funByFunCode: make(map[uint16]*funDescriptor),
	}
	numEmbeddedShort int
	numEmbeddedLong  int
	numExtended      int
)

func init() {
	embedShort("_slice", 3, getEvalFun(evalSlice))
	embedShort("_equal", 2, getEvalFun(evalEqual))
	embedShort("_len8", 1, getEvalFun(evalLen8))
	embedShort("_len16", 1, getEvalFun(evalLen16))
	embedShort("_not", 1, getEvalFun(evalNot))
	embedShort("_if", 3, getEvalFun(evalIf))
	embedShort("_isZero", 1, getEvalFun(evalIsZero))
	embedShort("_sum8", 2, getEvalFun(evalMustSum8))
	embedShort("_sum8_16", 2, getEvalFun(evalSum8_16))
	embedShort("_sum16", 2, getEvalFun(evalMustSum16))
	embedShort("_sum16_32", 2, getEvalFun(evalSum16_32))
	embedShort("_sum32", 2, getEvalFun(evalMustSum32))
	embedShort("_sum32_64", 2, getEvalFun(evalSum32_64))
	embedShort("_sum64", 2, getEvalFun(evalMustSum64))
	// argument access
	embedShort("$0", 0, getArgFun(0))
	embedShort("$1", 0, getArgFun(1))
	embedShort("$2", 0, getArgFun(2))
	embedShort("$3", 0, getArgFun(3))
	embedShort("$4", 0, getArgFun(4))
	embedShort("$5", 0, getArgFun(5))
	embedShort("$6", 0, getArgFun(6))
	embedShort("$7", 0, getArgFun(7))
	embedShort("$8", 0, getArgFun(8))
	embedShort("$9", 0, getArgFun(9))
	embedShort("$10", 0, getArgFun(10))
	embedShort("$11", 0, getArgFun(11))
	embedShort("$12", 0, getArgFun(12))
	embedShort("$13", 0, getArgFun(13))
	embedShort("$14", 0, getArgFun(14))
	embedShort("$15", 0, getArgFun(15))
	// context access
	embedShort("_data", 0, getEvalFun(evalData))
	embedShort("_path", 0, getEvalFun(evalPath))
	embedShort("_atPath", 1, getEvalFun(evalAtPath))
	// stateless varargs
	embedLong("concat", -1, getEvalFun(evalConcat))
	embedLong("and", -1, getEvalFun(evalAnd))
	embedLong("or", -1, getEvalFun(evalOr))
	embedLong("blake2b", -1, getEvalFun(evalBlake2b))
	// special
	embedLong("validSignature", 3, nil)

	fmt.Printf(`EasyFL function library:
    number of short embedded: %d out of max %d
    number of long embedded: %d out of max %d
    number of extended: %d out of max %d
`,
		numEmbeddedShort, easyfl.MaxNumEmbeddedShort, numEmbeddedLong, easyfl.MaxNumEmbeddedLong, numExtended, easyfl.MaxNumExtended)
}

func embedShort(sym string, requiredNumPar int, evalFun easyfl.EvalFunction) {
	if numEmbeddedShort >= easyfl.MaxNumEmbeddedShort {
		panic("too many embedded short functions")
	}
	mustUniqueName(sym)
	if requiredNumPar > 15 {
		panic("can't be more than 15 parameters")
	}
	dscr := &funDescriptor{
		sym:               sym,
		funCode:           uint16(numEmbeddedShort),
		requiredNumParams: requiredNumPar,
		evalFun:           evalFun,
	}
	Library.funByName[sym] = dscr
	Library.funByFunCode[dscr.funCode] = dscr
	numEmbeddedShort++
}

func embedLong(sym string, requiredNumPar int, evalFun easyfl.EvalFunction) {
	if numEmbeddedLong >= easyfl.MaxNumEmbeddedLong {
		panic("too many embedded long functions")
	}
	mustUniqueName(sym)
	if requiredNumPar > 15 {
		panic("can't be more than 15 parameters")
	}
	dscr := &funDescriptor{
		sym:               sym,
		funCode:           uint16(numEmbeddedLong + easyfl.FirstEmbeddedLongFun),
		requiredNumParams: requiredNumPar,
		evalFun:           evalFun,
	}
	Library.funByName[sym] = dscr
	Library.funByFunCode[dscr.funCode] = dscr
	numEmbeddedLong++
}

func extendLibrary(sym string, requiredNumPar int, evalFun easyfl.EvalFunction) error {
	if numExtended >= easyfl.MaxNumExtended {
		panic("too many extended functions")
	}
	if Library.ExistsFunction(sym) {
		return errors.New("repeating symbol '" + sym + "'")
	}
	if requiredNumPar > 15 {
		return errors.New("can't be more than 15 parameters")
	}
	dscr := &funDescriptor{
		sym:               sym,
		funCode:           uint16(numExtended + easyfl.FirstExtendedFun),
		requiredNumParams: requiredNumPar,
		evalFun:           evalFun,
	}
	Library.funByName[sym] = dscr
	Library.funByFunCode[dscr.funCode] = dscr
	numExtended++
	return nil
}

func mustUniqueName(sym string) {
	if Library.ExistsFunction(sym) {
		panic("repeating symbol '" + sym + "'")
	}
}

func getEvalFun(f runnerFunc) easyfl.EvalFunction {
	return func(glb easyfl.EvalContext) []byte {
		return f(glb.(*RunContext))
	}
}

func getExtendFun(f runnerFunc) easyfl.EvalFunction {
	return func(glb easyfl.EvalContext) []byte {
		g := glb.(*RunContext)

		g.pushCallBaseline()
		defer g.popCallBaseline()

		return f(g)
	}
}

func getArgFun(n byte) easyfl.EvalFunction {
	if n > 15 {
		panic("getArgFun: can be > 15")
	}
	return func(glb easyfl.EvalContext) []byte {
		return glb.(*RunContext).callArg(n)
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

func mustArithmArgs(glb *RunContext, bytesSize int) ([]byte, []byte) {
	a0 := glb.arg(0)
	a1 := glb.arg(1)
	if len(a0) != bytesSize || len(a1) != bytesSize {
		panic(fmt.Errorf("%d-bytes size parameters expected", bytesSize))
	}
	return a0, a1
}

func evalSum8_16(glb *RunContext) []byte {
	a0, a1 := mustArithmArgs(glb, 1)
	sum := uint16(a0[0]) + uint16(a1[0])
	ret := make([]byte, 2)
	binary.BigEndian.PutUint16(ret, sum)
	return ret
}

func evalMustSum8(glb *RunContext) []byte {
	a0, a1 := mustArithmArgs(glb, 1)
	sum := int(a0[0]) + int(a1[0])
	if sum > 255 {
		panic("_mustSum8: arithmetic overflow")
	}
	return []byte{byte(sum)}
}

func evalSum16_32(glb *RunContext) []byte {
	a0, a1 := mustArithmArgs(glb, 2)
	sum := uint32(binary.BigEndian.Uint16(a0)) + uint32(binary.BigEndian.Uint16(a1))
	ret := make([]byte, 4)
	binary.BigEndian.PutUint32(ret, sum)
	return ret
}

func evalMustSum16(glb *RunContext) []byte {
	a0, a1 := mustArithmArgs(glb, 2)
	sum := uint32(binary.BigEndian.Uint16(a0)) + uint32(binary.BigEndian.Uint16(a1))
	if sum > math.MaxUint16 {
		panic("_mustSum16: arithmetic overflow")
	}
	ret := make([]byte, 2)
	binary.BigEndian.PutUint16(ret, uint16(sum))
	return ret
}

func evalSum32_64(glb *RunContext) []byte {
	a0, a1 := mustArithmArgs(glb, 4)
	sum := uint64(binary.BigEndian.Uint32(a0)) + uint64(binary.BigEndian.Uint32(a1))
	ret := make([]byte, 8)
	binary.BigEndian.PutUint64(ret, sum)
	return ret
}

func evalMustSum32(glb *RunContext) []byte {
	a0, a1 := mustArithmArgs(glb, 4)
	sum := uint64(binary.BigEndian.Uint32(a0)) + uint64(binary.BigEndian.Uint32(a1))
	if sum > math.MaxUint32 {
		panic("_mustSum32: arithmetic overflow")
	}
	ret := make([]byte, 4)
	binary.BigEndian.PutUint32(ret, uint32(sum))
	return ret
}

func evalMustSum64(glb *RunContext) []byte {
	a0, a1 := mustArithmArgs(glb, 8)
	s0 := binary.BigEndian.Uint64(a0)
	s1 := binary.BigEndian.Uint64(a1)
	if s0 > math.MaxUint64-s1 {
		panic("_mustSum64: arithmetic overflow")
	}
	ret := make([]byte, 8)
	binary.BigEndian.PutUint64(ret, s0+s1)
	return ret
}
