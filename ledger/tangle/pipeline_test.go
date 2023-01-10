package tangle

import (
	"testing"
	"time"

	"github.com/lunfardo314/easyutxo/util/testutil"
)

func TestPipelineBasic(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		log := testutil.NewSimpleLogger(true)
		pipe := NewPipeline(log)
		pipe.Start()
		time.Sleep(10 * time.Millisecond)
		pipe.Stop()
		time.Sleep(10 * time.Millisecond)
		log.Sync()
	})
}
