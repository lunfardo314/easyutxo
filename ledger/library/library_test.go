package library

import (
	"encoding/binary"
	"testing"

	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/stretchr/testify/require"
)

const formula1 = "def unlockBlock(0) = _atPath(concat(0x0000, _slice(_path, 2, 5)))"

func TestParse(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		ret, err := easyfl.ParseFunctions(formula1)
		require.NoError(t, err)
		require.NotNil(t, ret)
	})
	t.Run("2", func(t *testing.T) {
		ret, err := easyfl.ParseFunctions(SigLockConstraint)
		require.NoError(t, err)
		require.NotNil(t, ret)
	})
	t.Run("3", func(t *testing.T) {
		ret, err := easyfl.ParseFunctions(formula1)
		require.NoError(t, err)
		require.EqualValues(t, 1, len(ret))

		code, err := easyfl.FormulaSourceToBinary(Library, ret[0].NumParams, ret[0].SourceCode)
		require.NoError(t, err)
		t.Logf("code len: %d", len(code))
	})
	t.Run("4", func(t *testing.T) {
		parsed, err := easyfl.ParseFunctions(SigLockConstraint)
		require.NoError(t, err)
		err = Library.compileAndAddMany(parsed)
		require.NoError(t, err)
	})
	t.Run("5", func(t *testing.T) {
		parsed, err := easyfl.ParseFunctions(formula1)
		require.NoError(t, err)
		require.EqualValues(t, 1, len(parsed))

		code, err := easyfl.FormulaSourceToBinary(Library, parsed[0].NumParams, parsed[0].SourceCode)
		require.NoError(t, err)
		t.Logf("code len: %d", len(code))

		f, err := easyfl.FormulaTreeFromBinary(Library, code)
		require.NoError(t, err)
		require.NotNil(t, f)
	})
}

func TestEval(t *testing.T) {
	runTest := func(s string, path []byte) []byte {
		f, code, err := easyfl.CompileFormula(Library, 0, s)
		require.NoError(t, err)

		ctx := NewRunContext(lazyslice.TreeEmpty(), path)
		ret := ctx.Eval(f)
		t.Logf("code len: %d, result: %v -- '%s'", len(code), ret, s)
		return ret
	}
	t.Run("1", func(t *testing.T) {
		path := lazyslice.Path(0, 2)
		res := runTest("_path", path)
		require.EqualValues(t, path, res)
	})
	t.Run("2", func(t *testing.T) {
		path := lazyslice.Path(1, 2, 1)
		res := runTest("_len8(_path)", path)
		require.EqualValues(t, []byte{3}, res)
	})
	t.Run("3", func(t *testing.T) {
		res := runTest("concat(1,2,3,4,5)", nil)
		require.EqualValues(t, []byte{1, 2, 3, 4, 5}, res)
	})
	t.Run("4", func(t *testing.T) {
		res := runTest("concat(concat(1,2),concat(3,4,5))", nil)
		require.EqualValues(t, []byte{1, 2, 3, 4, 5}, res)
	})
	t.Run("5", func(t *testing.T) {
		res := runTest("_slice(concat(concat(1,2),concat(3,4,5)),2,4)", nil)
		require.EqualValues(t, []byte{3, 4}, res)
	})
	t.Run("6", func(t *testing.T) {
		path := lazyslice.Path(1, 2, 1)
		res := runTest("_if(_equal(_len8(_path),3), 0x01, 0x05)", path)
		require.EqualValues(t, []byte{1}, res)
	})
	t.Run("7", func(t *testing.T) {
		path := lazyslice.Path(1, 2)
		res := runTest("_if(_equal(_len8(_path),3), 0x01, 0x05)", path)
		require.EqualValues(t, []byte{5}, res)
	})
	t.Run("8", func(t *testing.T) {
		path := lazyslice.Path(1, 2, 1)
		res := runTest("_if(_not(_equal(_len8(_path),3)), 0x01, 0x0506)", path)
		require.EqualValues(t, []byte{5, 6}, res)
	})
	t.Run("9", func(t *testing.T) {
		res := runTest("15", nil)
		require.EqualValues(t, []byte{15}, res)
	})
	t.Run("10", func(t *testing.T) {
		res := runTest("concat()", nil)
		require.EqualValues(t, 0, len(res))
	})
	t.Run("11", func(t *testing.T) {
		res := runTest("u16/256", nil)
		require.EqualValues(t, []byte{0x01, 0x00}, res)
	})
	t.Run("12", func(t *testing.T) {
		res := runTest("u32/70000", nil)
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], 70000)
		require.EqualValues(t, b[:], res)
	})
	t.Run("14", func(t *testing.T) {
		res := runTest("u64/10000000000", nil)
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], 10000000000)
		require.EqualValues(t, b[:], res)
	})
	t.Run("15", func(t *testing.T) {
		res := runTest("_isZero(0x000000)", nil)
		require.True(t, len(res) > 0)
	})
	t.Run("16", func(t *testing.T) {
		res := runTest("_isZero(0x002000)", nil)
		require.True(t, len(res) == 0)
	})
	t.Run("17", func(t *testing.T) {
		res := runTest("_sum8_16(100, 160)", nil)
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], 260)
		require.EqualValues(t, b[:], res)
	})
}
