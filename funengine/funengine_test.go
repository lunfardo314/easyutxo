package funengine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const formula1 = "def unlockBlock(0) = _atPath(concat(0x0000,_slice(_path, 2, 5)))"

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

		err = ret[0].genCode(make(map[string]*funDef))
		require.NoError(t, err)
		t.Logf("code len: %d", len(ret[0].code))
	})
	t.Run("4", func(t *testing.T) {
		lib, err := compileToLibrary(sigLockConstraint, FirstUserFunCode)
		require.NoError(t, err)
		require.True(t, len(lib) > 0)
	})
	t.Run("5", func(t *testing.T) {
		ret, err := parseDefinitions(formula1)
		require.NoError(t, err)
		require.EqualValues(t, 1, len(ret))

		err = ret[0].genCode(make(map[string]*funDef))
		require.NoError(t, err)
		t.Logf("code len: %d", len(ret[0].code))
		rdr := NewCodeReader(ret[0].code, make(map[uint16]*funDef))
		count := 0
		for c := rdr.MustNext(); c != nil; c = rdr.MustNext() {
			count++
		}
		t.Logf("number of calls: %d", count)
	})
}
