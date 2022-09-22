package ledger

import (
	"testing"

	"github.com/lunfardo314/easyutxo/easyfl"
	"github.com/stretchr/testify/require"
)

func TestConstraints(t *testing.T) {
	t.Run("2", func(t *testing.T) {
		ret, err := easyfl.ParseFunctions(SigLockConstraint)
		require.NoError(t, err)
		require.NotNil(t, ret)
	})

}
