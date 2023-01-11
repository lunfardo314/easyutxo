package tangle

import (
	"time"

	"github.com/lunfardo314/easyutxo/ledger/state"
	"github.com/lunfardo314/easyutxo/util/fifoqueue"
	"github.com/lunfardo314/easyutxo/util/waitingroom"
	"go.uber.org/zap"
)

type Pipeline struct {
	log               *zap.SugaredLogger
	waitingRoom       *waitingroom.WaitingRoom
	preValidatorQueue *fifoqueue.FIFOQueue[[]byte]
	solidifierQueue   *fifoqueue.FIFOQueue[*state.Transaction]
}

func NewPipeline(globalLog *zap.SugaredLogger) *Pipeline {
	return &Pipeline{
		log:               globalLog,
		waitingRoom:       waitingroom.Create(1 * time.Second),
		preValidatorQueue: fifoqueue.New[[]byte](),
		solidifierQueue:   fifoqueue.New[*state.Transaction](),
	}
}

func (pi *Pipeline) Start() {
	pi.startDaemon("[prevalid]", pi.prevalidate, func(log *zap.SugaredLogger) {
		// close downstream
		log.Infof("closing downstream solidifier queue")
		pi.solidifierQueue.Close()
	})

	pi.startDaemon("[solidifier]", pi.solidify)
}

func (pi *Pipeline) startDaemon(name string, consumeFun func(log *zap.SugaredLogger), onClose ...func(log *zap.SugaredLogger)) {
	go func() {
		log := pi.log.Named(name)
		log.Infof("STARTED")
		consumeFun(log)
		log.Infof("STOPPED")
		if len(onClose) > 0 && onClose[0] != nil {
			onClose[0](log)
		}
	}()
}

func (pi *Pipeline) Stop() {
	pi.waitingRoom.Stop()
	pi.preValidatorQueue.Close()
}

func (pi *Pipeline) ProcessTransaction(txBytes []byte) {
	pi.preValidatorQueue.Write(txBytes)
}

func (pi *Pipeline) finalStateTimestamp() uint32 {
	return 0
}

const maxWaitingSeconds = 10

func (pi *Pipeline) maxSecondsInTheFuture() uint32 {
	return maxWaitingSeconds
}

func (pi *Pipeline) prevalidate(log *zap.SugaredLogger) {
	pi.preValidatorQueue.Consume(func(txBytes []byte) {
		var tx *state.Transaction

		nowis := uint32(time.Now().Unix())
		tsLowerBound := pi.finalStateTimestamp() + 1
		tsUpperBound := nowis + pi.maxSecondsInTheFuture()
		tx, err := state.TransactionFromTransferableBytes(txBytes, tsLowerBound, tsUpperBound)
		if err != nil {
			log.Debugf("transaction bytes dropped. Reason: '%v'", err)
			return
		}

		txid := tx.ID()
		if tx.Timestamp() <= nowis {
			// timestamp is in the past, pass it to the solidifier
			pi.solidifierQueue.Write(tx)
			log.Debugf("to solidifier: ID = %s", txid.String())
		} else {
			// timestamp is in the future. Put it into the waiting room
			log.Debugf("to waiting room: ID = %s", txid.String())
			pi.waitingRoom.WaitUntilUnixSec(tx.Timestamp(), func() {
				pi.solidifierQueue.Write(tx)
				log.Debugf("transaction OUT from the waiting room: ID = %s", txid.String())
			})
		}
	})
}

func (pi *Pipeline) solidify(log *zap.SugaredLogger) {
	pi.solidifierQueue.Consume(func(tx *state.Transaction) {
		txid := tx.ID()
		log.Debugf("trasaction IN: ID = %s", txid.String())
	})
}
