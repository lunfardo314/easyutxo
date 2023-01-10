package tangle

import (
	"github.com/lunfardo314/easyutxo/ledger/state"
	"github.com/lunfardo314/easyutxo/util/fifoqueue"
	"github.com/lunfardo314/unitrie/common"
	"go.uber.org/zap"
)

type Pipeline struct {
	log          *zap.SugaredLogger
	preValidator *fifoqueue.FIFOQueue[[]byte]
	solidifier   *fifoqueue.FIFOQueue[*state.Transaction]
}

func NewPipeline(globalLog *zap.SugaredLogger) *Pipeline {
	return &Pipeline{
		log:          globalLog,
		preValidator: fifoqueue.NewFIFOQueue[[]byte](),
		solidifier:   fifoqueue.NewFIFOQueue[*state.Transaction](),
	}
}

func (pipe *Pipeline) Start() {
	go func() {
		log := pipe.log.Named("preValidator")
		log.Infof("STARTED")

		pipe.preValidator.Consume(func(txBytes []byte) {
			var tx *state.Transaction

			err := common.CatchPanicOrError(func() error {
				var err1 error
				tx, err1 = state.MustTransactionFromTransferableBytes(txBytes)
				return err1
			})
			if err != nil {
				log.Debugf("transaction bytes dropped. Reason: '%v'", err)
				return
			}
			pipe.solidifier.Write(tx)
			txid := tx.ID()
			log.Debugf("transaction OUT: ID = %s", txid.String())
		})

		// close downstream
		pipe.solidifier.Close()
		log.Infof("STOPPED")
	}()

	go func() {
		log := pipe.log.Named("solidifier")
		log.Infof("STARTED")
		pipe.solidifier.Consume(func(tx *state.Transaction) {

			txid := tx.ID()
			log.Debugf("trasaction IN: ID = %s", txid.String())
		})
		log.Infof("STOPPED")
	}()
}

func (pipe *Pipeline) Stop() {
	pipe.preValidator.Close()
}

func (pipe *Pipeline) ProcessTransaction(txBytes []byte) {
	pipe.preValidator.Write(txBytes)
}
