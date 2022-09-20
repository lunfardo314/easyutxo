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
