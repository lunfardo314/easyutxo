package funengine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const formula1 = "def unlockBlock(0) = bytesAtPath(concat(bytes(0,0),slice(path, 2, 5)))"

func TestParse(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		ret, err := parseDefinitions(formula1)
		require.NoError(t, err)
		require.NotNil(t, ret)
	})
	t.Run("2", func(t *testing.T) {
		ret, err := parseDefinitions(sigLockConstraint)
		require.NoError(t, err)
		require.NotNil(t, ret)
	})
	t.Run("3", func(t *testing.T) {
		ret, err := parseDefinitions(formula1)
		require.NoError(t, err)
		require.EqualValues(t, 1, len(ret))

		err = ret[0].genCode(functionsBySymbol)
		require.NoError(t, err)
		t.Logf("code len: %d", len(ret[0].code))
	})
	t.Run("4", func(t *testing.T) {
		err := compileToLibrary(sigLockConstraint, functionsBySymbol)
		require.NoError(t, err)
	})
}
