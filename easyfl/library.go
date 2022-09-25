package easyfl

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

type funDescriptor struct {
	sym               string
	funCode           uint16
	requiredNumParams int
	evalFun           EvalFunction
}

type libraryData struct {
	funByName    map[string]*funDescriptor
	funByFunCode map[uint16]*funDescriptor
}

var (
	theLibrary = &libraryData{
		funByName:    make(map[string]*funDescriptor),
		funByFunCode: make(map[uint16]*funDescriptor),
	}
	numEmbeddedShort = EmbeddedReservedUntil + 1
	numEmbeddedLong  int
	numExtended      int
)

func init() {
	// basic
	EmbedShort("slice", 3, evalSlice)
	EmbedShort("equal", 2, evalEqual)
	EmbedShort("len8", 1, evalLen8)
	EmbedShort("len16", 1, evalLen16)
	EmbedShort("not", 1, evalNot)
	EmbedShort("if", 3, evalIf)
	EmbedShort("isZero", 1, evalIsZero)
	// stateless varargs
	EmbedLong("concat", -1, evalConcat)
	EmbedLong("and", -1, evalAnd)
	EmbedLong("or", -1, evalOr)

	// safe arithmetics
	EmbedShort("sum8", 2, evalMustSum8)
	EmbedShort("sum8_16", 2, evalSum8_16)
	EmbedShort("sum16", 2, evalMustSum16)
	EmbedShort("sum16_32", 2, evalSum16_32)
	EmbedShort("sum32", 2, evalMustSum32)
	EmbedShort("sum32_64", 2, evalSum32_64)
	EmbedShort("sum64", 2, evalMustSum64)
	EmbedShort("sub8", 2, evalMustSub8)
	// comparison
	EmbedShort("lessThan", 2, evalLessThan)
	Extend("lessOrEqualThan", "or(lessThan($0,$1),equal($0,$1))")
	Extend("greaterThan", "not(lessOrEqualThan($0,$1))")
	Extend("greaterOrEqualThan", "not(lessThan($0,$1))")
	// other
	Extend("nil", "or()")
	Extend("tail", "slice($0, $1, sub8(len8($0),1))")

	fmt.Printf(`EasyFL function library:
    number of short embedded: %d out of max %d
    number of long embedded: %d out of max %d
    number of extended: %d out of max %d
`,
		numEmbeddedShort, MaxNumEmbeddedShort, numEmbeddedLong, MaxNumEmbeddedLong, numExtended, MaxNumExtended)
}

func EmbedShort(sym string, requiredNumPar int, evalFun EvalFunction) uint16 {
	if numEmbeddedShort >= MaxNumEmbeddedShort {
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
	theLibrary.funByName[sym] = dscr
	theLibrary.funByFunCode[dscr.funCode] = dscr
	numEmbeddedShort++

	return dscr.funCode
}

func EmbedLong(sym string, requiredNumPar int, evalFun EvalFunction) uint16 {
	if numEmbeddedLong >= MaxNumEmbeddedLong {
		panic("too many embedded long functions")
	}
	mustUniqueName(sym)
	if requiredNumPar > 15 {
		panic("can't be more than 15 parameters")
	}
	dscr := &funDescriptor{
		sym:               sym,
		funCode:           uint16(numEmbeddedLong + FirstEmbeddedLongFun),
		requiredNumParams: requiredNumPar,
		evalFun:           evalFun,
	}
	theLibrary.funByName[sym] = dscr
	theLibrary.funByFunCode[dscr.funCode] = dscr
	numEmbeddedLong++

	return dscr.funCode
}

func Extend(sym string, source string) uint16 {
	f, numParam, _, err := CompileFormula(source)
	if err != nil {
		panic(fmt.Errorf("error while compiling '%s': %v", sym, err))
	}

	if numExtended >= MaxNumExtended {
		panic("too many extended functions")
	}
	if existsFunction(sym) {
		panic(errors.New("repeating symbol '" + sym + "'"))
	}
	if numParam > 15 {
		panic(errors.New("can't be more than 15 parameters"))
	}
	dscr := &funDescriptor{
		sym:               sym,
		funCode:           uint16(numExtended + FirstExtendedFun),
		requiredNumParams: numParam,
		evalFun: func(ctx *RunContext) []byte {
			return ctx.Eval(f)
		},
	}
	theLibrary.funByName[sym] = dscr
	theLibrary.funByFunCode[dscr.funCode] = dscr
	numExtended++
	return dscr.funCode
}

func mustUniqueName(sym string) {
	if existsFunction(sym) {
		panic("repeating symbol '" + sym + "'")
	}
}

func extendWithMany(source string) error {
	parsed, err := ParseFunctions(source)
	if err != nil {
		return err
	}
	for _, pf := range parsed {
		Extend(pf.Sym, pf.SourceCode)
	}
	return nil
}

func MustExtendWithMany(source string) {
	if err := extendWithMany(source); err != nil {
		panic(err)
	}
}

func existsFunction(sym string) bool {
	_, found := theLibrary.funByName[sym]
	return found
}

func functionByName(sym string) (*funInfo, error) {
	fd, found := theLibrary.funByName[sym]
	if !found {
		return nil, fmt.Errorf("no such function in the library: '%s'", sym)
	}
	ret := &funInfo{
		Sym:       sym,
		FunCode:   fd.funCode,
		NumParams: fd.requiredNumParams,
	}
	switch {
	case fd.funCode < FirstEmbeddedLongFun:
		ret.IsEmbedded = true
		ret.IsShort = true
	case fd.funCode < FirstExtendedFun:
		ret.IsEmbedded = true
		ret.IsShort = false
	}
	return ret, nil
}

func functionByCode(funCode uint16) (EvalFunction, int, error) {
	var libData *funDescriptor
	libData = theLibrary.funByFunCode[funCode]
	if libData == nil {
		return nil, 0, fmt.Errorf("wrong function code %d", funCode)
	}
	return libData.evalFun, libData.requiredNumParams, nil
}

func evalSlice(glb *RunContext) []byte {
	data := glb.Arg(0)
	from := glb.Arg(1)
	to := glb.Arg(2)
	return data[from[0]:to[0]]
}

func evalEqual(glb *RunContext) []byte {
	if bytes.Equal(glb.Arg(0), glb.Arg(1)) {
		return []byte{0xff}
	}
	return nil
}

func evalLen8(glb *RunContext) []byte {
	sz := len(glb.Arg(0))
	if sz > math.MaxUint8 {
		panic("len8: size of the data > 255")
	}
	return []byte{byte(sz)}
}

func evalLen16(glb *RunContext) []byte {
	data := glb.Arg(0)
	if len(data) > math.MaxUint16 {
		panic("len16: size of the data > uint16")
	}
	var ret [2]byte
	binary.BigEndian.PutUint16(ret[:], uint16(len(data)))
	return ret[:]
}

func evalIf(glb *RunContext) []byte {
	cond := glb.Arg(0)
	if len(cond) != 0 {
		// true
		return glb.Arg(1)
	}
	return glb.Arg(2)
}

func evalIsZero(glb *RunContext) []byte {
	for _, b := range glb.Arg(0) {
		if b != 0 {
			return nil
		}
	}
	return []byte{0xff}
}

func evalNot(glb *RunContext) []byte {
	if len(glb.Arg(0)) == 0 {
		return []byte{0xff}
	}
	return nil
}

func evalConcat(glb *RunContext) []byte {
	var buf bytes.Buffer
	for i := byte(0); i < glb.Arity(); i++ {
		buf.Write(glb.Arg(i))
	}
	return buf.Bytes()
}

func evalAnd(glb *RunContext) []byte {
	for i := byte(0); i < glb.Arity(); i++ {
		if len(glb.Arg(i)) == 0 {
			return nil
		}
	}
	return []byte{0xff}
}

func evalOr(glb *RunContext) []byte {
	for i := byte(0); i < glb.Arity(); i++ {
		if len(glb.Arg(i)) != 0 {
			return []byte{0xff}
		}
	}
	return nil
}

func mustArithmArgs(glb *RunContext, bytesSize int) ([]byte, []byte) {
	a0 := glb.Arg(0)
	a1 := glb.Arg(1)
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

func evalMustSub8(glb *RunContext) []byte {
	a0, a1 := mustArithmArgs(glb, 1)
	if a0[0] < a1[0] {
		panic("_mustSub8: underflow in subtraction")
	}
	return []byte{a0[0] - a1[0]}
}

// lexicographical comparison of two slices of equal length
func evalLessThan(glb *RunContext) []byte {
	a0 := glb.Arg(0)
	a1 := glb.Arg(1)
	if len(a0) != len(a1) {
		panic("evalLessThan: operands must be equal length")
	}
	for i := range a0 {
		if a0[i] < a1[i] {
			return []byte{0xff} // true
		}
	}
	return nil
}
