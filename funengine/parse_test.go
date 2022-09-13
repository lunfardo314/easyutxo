package funengine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		err := parse(sigLockConstraint)
		require.NoError(t, err)
	})
}
