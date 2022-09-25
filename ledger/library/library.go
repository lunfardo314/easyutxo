package library

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/lunfardo314/easyutxo/ledger/library"
	"golang.org/x/crypto/blake2b"
)

func init() {
	// context access
	embedShort("@", 0, evalPath)
	embedShort("atPath", 1, evalAtPath)
	embedLong("blake2b", -1, evalBlake2b)
	// special transaction related
	embedLong("validSignatureED25519", 3, evalValidSigED25519)

	MustExtendLibrary("nil", "or()")
	MustExtendLibrary("tail", "slice($0, $1, sub8(len8($0),1))")

	MustExtendLibrary("lessOrEqualThan", "or(lessThan($0,$1),equal($0,$1))")
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
		numEmbeddedShort, MaxNumEmbeddedShort, numEmbeddedLong, MaxNumEmbeddedLong, numExtended, MaxNumExtended)
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

func evalPath(glb *RunContext) []byte {
	return glb.Global().(*library.GlobalContext).invocationPath
}

func evalAtPath(glb *RunContext) []byte {
	return glb.Global().(*library.GlobalContext).dataTree.BytesAtPath(glb.Arg(0))
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

func evalBlake2b(glb *RunContext) []byte {
	ret := blake2b.Sum256(evalConcat(glb))
	return ret[:]
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

func evalValidSigED25519(glb *RunContext) []byte {
	essence := glb.Arg(0)
	signature := glb.Arg(1)
	pubKey := glb.Arg(2)

	if ed25519.Verify(pubKey, essence, signature) {
		return []byte{0xff}
	}
	return nil
}

func evalRequireAll(glb *RunContext) []byte {
	blockIndices := glb.Arg(0)
	path := glb.Global().(*library.GlobalContext).invocationPath
	myIdx := path[len(path)-1]
	pathCopy := make([]byte, len(path))
	copy(pathCopy, path)

	for _, idx := range blockIndices {
		if idx <= myIdx {
			// only forward
			panic("evalRequireAll: can only invoke constraints forward")
		}
		pathCopy[len(path)-1] = idx
		if len(library.invokeConstraint(glb.Global().(*library.GlobalContext).dataTree, path)) == 0 {
			return nil
		}
	}
	return []byte{0xff}
}

func evalRequireAny(glb *RunContext) []byte {
	blockIndices := glb.Arg(0)
	path := glb.Global().(*library.GlobalContext).invocationPath
	myIdx := path[len(path)-1]
	pathCopy := make([]byte, len(path))
	copy(pathCopy, path)

	for _, idx := range blockIndices {
		if idx <= myIdx {
			// only forward
			panic("evalRequireAll: can only invoke constraints forward")
		}
		path[len(path)-1] = idx
		if len(library.invokeConstraint(glb.Global().(*library.GlobalContext).dataTree, path)) != 0 {
			return []byte{0xff}
		}
	}
	return nil
}
