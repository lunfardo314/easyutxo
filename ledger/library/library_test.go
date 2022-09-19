package library

import (
	"testing"

	"github.com/lunfardo314/easyutxo/easyfl"
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

		code, err := easyfl.CompileFormula(Library, ret[0].NumParams, ret[0].SourceCode)
		require.NoError(t, err)
		t.Logf("code len: %d", len(code))
	})
	t.Run("5", func(t *testing.T) {
		parsed, err := easyfl.ParseFunctions(formula1)
		require.NoError(t, err)
		require.EqualValues(t, 1, len(parsed))

		code, err := easyfl.CompileFormula(Library, parsed[0].NumParams, parsed[0].SourceCode)
		require.NoError(t, err)
		t.Logf("code len: %d", len(code))
		ctx := NewRunContext(nil, nil)
		rdr := easyfl.NewCodeReader(ctx, code)
		countCall := 0
		countData := 0
		for c := rdr.MustNext(); c != nil; c = rdr.MustNext() {
			switch c.(type) {
			case []byte:
				countData++
			default:
				countCall++
			}
		}
		t.Logf("number of calls: %d, number of data: %d", countCall, countData)
		require.EqualValues(t, 4, countCall)
		require.EqualValues(t, 3, countData)
	})

	t.Run("4", func(t *testing.T) {
		parsed, err := easyfl.ParseFunctions(SigLockConstraint)
		require.NoError(t, err)
		err = Library.compileAndAddMany(parsed)
		require.NoError(t, err)
	})
}
