package library

import (
	"bytes"
	"crypto/ed25519"
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
	numEmbeddedShort   int
	numEmbeddedLong    int
	numExtended        int
	FuncCodeRequireAll uint16
	FuncCodeRequireAny uint16
)

func init() {
	embedShort("slice", 3, ev(evalSlice))
	embedShort("equal", 2, ev(evalEqual))
	embedShort("len8", 1, ev(evalLen8))
	embedShort("len16", 1, ev(evalLen16))
	embedShort("not", 1, ev(evalNot))
	embedShort("if", 3, ev(evalIf))
	embedShort("isZero", 1, ev(evalIsZero))

	// the two codes needed explicitly for the construction of the output
	FuncCodeRequireAll = embedShort("requireAll", 1, ev(evalRequireAll))
	FuncCodeRequireAny = embedShort("requireAny", 1, ev(evalRequireAny))
	// safe arithmetics
	embedShort("sum8", 2, ev(evalMustSum8))
	embedShort("sum8_16", 2, ev(evalSum8_16))
	embedShort("sum16", 2, ev(evalMustSum16))
	embedShort("sum16_32", 2, ev(evalSum16_32))
	embedShort("sum32", 2, ev(evalMustSum32))
	embedShort("sum32_64", 2, ev(evalSum32_64))
	embedShort("sum64", 2, ev(evalMustSum64))
	embedShort("sub8", 2, ev(evalMustSub8))
	// comparison
	embedShort("lessThan", 2, ev(evalLessThan))
	embedShort("lessOrEqualThan", 2, ev(evalLessOrEqualThan))
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
	embedShort("@", 0, ev(evalPath))
	embedShort("atPath", 1, ev(evalAtPath))
	// stateless varargs
	embedLong("concat", -1, ev(evalConcat))
	embedLong("and", -1, ev(evalAnd))
	embedLong("or", -1, ev(evalOr))

	embedLong("blake2b", -1, ev(evalBlake2b))
	// special transaction related
	embedLong("validSignatureED25519", 3, ev(evalValidSigED25519))

	MustExtendLibrary("nil", "or()")
	MustExtendLibrary("tail", "slice($0, $1, sub8(len8($0),1))")

	MustExtendLibrary("greaterThan", "not(lessOrEqualThan($0,$1))")
	MustExtendLibrary("greaterOrEqualThan", "not(lessThan($0,$1))")

	MustExtendLibrary("txBytes", "atPath(0x00)")
	MustExtendLibrary("txInputIDsBytes", "atPath(0x0001)")
	MustExtendLibrary("txOutputBytes", "atPath(0x0002)")
	MustExtendLibrary("txTimestampBytes", "atPath(0x0003)")
	MustExtendLibrary("txInputCommitmentBytes", "atPath(0x0004)")
	MustExtendLibrary("txLocalLibBytes", "atPath(0x0005)")
	MustExtendLibrary("txid", "blake2b(txBytes)")
	MustExtendLibrary("txEssenceBytes", "concat(txInputIDsBytes, txOutputBytes, txTimestampBytes, txInputCommitmentBytes, txLocalLibBytes)")
	MustExtendLibrary("addrED25519FromPubKey", "blake2b($0)")

	MustExtendLibrary("selfConstraint", "atPath(@)")
	MustExtendLibrary("selfConstraintData", "if(equal(slice(selfConstraint,0,1), 0),nil,tail(selfConstraint,1))")
	MustExtendLibrary("selfOutputIndex", "slice(@,2,4)")
	MustExtendLibrary("selfUnlockBlock", "atPath(concat(0, 0, slice(@, 2, 5)))")
	MustExtendLibrary("selfReferencedConstraint", "atPath(concat(slice(@,0,2), selfUnlockBlock))")
	MustExtendLibrary("selfConsumedContext", "equal(slice(@,0,2), 0x0100)")
	MustExtendLibrary("selfOutputContext", "not(selfConsumedContext)")

	fmt.Printf(`EasyFL function library:
    number of short embedded: %d out of max %d
    number of long embedded: %d out of max %d
    number of extended: %d out of max %d
`,
		numEmbeddedShort, easyfl.MaxNumEmbeddedShort, numEmbeddedLong, easyfl.MaxNumEmbeddedLong, numExtended, easyfl.MaxNumExtended)
}

func embedShort(sym string, requiredNumPar int, evalFun easyfl.EvalFunction) uint16 {
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

	return dscr.funCode
}

func embedLong(sym string, requiredNumPar int, evalFun easyfl.EvalFunction) uint16 {
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

	return dscr.funCode
}

func ExtendLibrary(sym string, source string) (uint16, error) {
	f, numParam, _, err := easyfl.CompileFormula(Library, source)
	if err != nil {
		return 0, fmt.Errorf("error while compiling '%s': %v", sym, err)
	}

	if numExtended >= easyfl.MaxNumExtended {
		panic("too many extended functions")
	}
	if Library.ExistsFunction(sym) {
		return 0, errors.New("repeating symbol '" + sym + "'")
	}
	if numParam > 15 {
		return 0, errors.New("can't be more than 15 parameters")
	}
	dscr := &funDescriptor{
		sym:               sym,
		funCode:           uint16(numExtended + easyfl.FirstExtendedFun),
		requiredNumParams: numParam,
		evalFun: getExtendFun(func(ctx *RunContext) []byte {
			return ctx.Eval(f)
		}),
	}
	Library.funByName[sym] = dscr
	Library.funByFunCode[dscr.funCode] = dscr
	numExtended++
	return dscr.funCode, nil
}

func MustExtendLibrary(sym string, source string) uint16 {
	ret, err := ExtendLibrary(sym, source)
	if err != nil {
		panic(err)
	}
	return ret
}

func mustUniqueName(sym string) {
	if Library.ExistsFunction(sym) {
		panic("repeating symbol '" + sym + "'")
	}
}

func extendWitMany(source string) error {
	parsed, err := easyfl.ParseFunctions(source)
	if err != nil {
		return err
	}
	for _, pf := range parsed {
		_, err = ExtendLibrary(pf.Sym, pf.SourceCode)
		if err != nil {
			return err
		}
	}
	return nil
}

func MustExtendWithMany(source string) {
	if err := extendWitMany(source); err != nil {
		panic(err)
	}
}

func ev(f EvalFunc) easyfl.EvalFunction {
	return func(glb interface{}) []byte {
		return f(glb.(*RunContext))
	}
}

func getExtendFun(f EvalFunc) easyfl.EvalFunction {
	return func(glb interface{}) []byte {
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
	return func(glb interface{}) []byte {
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

func evalAtPath(glb *RunContext) []byte {
	return glb.dataTree.BytesAtPath(glb.arg(0))
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

func evalMustSub8(glb *RunContext) []byte {
	a0, a1 := mustArithmArgs(glb, 1)
	if a0[0] < a1[0] {
		panic("_mustSub8: underflow in subtraction")
	}
	return []byte{a0[0] - a1[0]}
}

func evalLessThan(glb *RunContext) []byte {
	a0 := glb.arg(0)
	a1 := glb.arg(1)
	if len(a0) != len(a1) {
		panic("evalLessThan: operands must be equal length")
	}
	allEqual := true
	for i := range a0 {
		if a0[i] > a1[1] {
			return nil // false
		}
		if a0[i] < a1[1] {
			allEqual = false
		}
	}
	if allEqual {
		return nil
	}
	return []byte{0xff}
}

func evalLessOrEqualThan(glb *RunContext) []byte {
	a0 := glb.arg(0)
	a1 := glb.arg(1)
	if len(a0) != len(a1) {
		panic("evalLessThan: operands must be equal length")
	}
	for i := range a0 {
		if a0[i] > a1[1] {
			return nil // false
		}
	}
	return []byte{0xff}
}

func evalValidSigED25519(glb *RunContext) []byte {
	essence := glb.arg(0)
	signature := glb.arg(1)
	pubKey := glb.arg(2)

	if ed25519.Verify(pubKey, essence, signature) {
		return []byte{0xff}
	}
	return nil
}

func evalRequireAll(glb *RunContext) []byte {
	blockIndices := glb.arg(0)
	myIdx := glb.invocationPath[len(glb.invocationPath)-1]
	path := make([]byte, len(glb.invocationPath))
	copy(path, glb.invocationPath)

	for _, idx := range blockIndices {
		if idx <= myIdx {
			// only forward
			panic("evalRequireAll: can only invoke constraints forward")
		}
		path[len(path)-1] = idx
		if len(invokeConstraint(glb.dataTree, path)) == 0 {
			return nil
		}
	}
	return []byte{0xff}
}

func evalRequireAny(glb *RunContext) []byte {
	blockIndices := glb.arg(0)
	myIdx := glb.invocationPath[len(glb.invocationPath)-1]
	path := make([]byte, len(glb.invocationPath))
	copy(path, glb.invocationPath)

	for _, idx := range blockIndices {
		if idx <= myIdx {
			// only forward
			panic("evalRequireAll: can only invoke constraints forward")
		}
		path[len(path)-1] = idx
		if len(invokeConstraint(glb.dataTree, path)) != 0 {
			return []byte{0xff}
		}
	}
	return nil
}
