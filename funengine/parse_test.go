package funengine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const formula1 = "def unlockBlock() = bytesAtPath(concat(bytes(0,0),slice(path(), 2, 5)))"

func TestParse(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		ret, err := parse(formula1)
		require.NoError(t, err)
		require.NotNil(t, ret)
	})
}
