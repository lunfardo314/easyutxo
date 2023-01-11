package waitingroom

import (
	"sync"
	"time"

	"github.com/lunfardo314/unitrie/common"
	"go.uber.org/atomic"
)

type WaitingRoom struct {
	mutex   sync.RWMutex
	d       map[time.Time][]func()
	period  time.Duration
	stopped atomic.Bool
}

var defaultPoolingPeriod = 1 * time.Second

func Create(poolEvery ...time.Duration) *WaitingRoom {
	ret := &WaitingRoom{
		d:      make(map[time.Time][]func()),
		period: defaultPoolingPeriod,
	}
	if len(poolEvery) > 0 {
		ret.period = poolEvery[0]
	}

	go ret.pooling()
	return ret
}

func (d *WaitingRoom) pooling() {
	for {
		lst := make([][]func(), 0)
		dels := make([]time.Time, 0)

		time.Sleep(d.period)

		if d.stopped.Load() {
			return
		}

		nowis := time.Now()

		for t, l := range d.d {
			if t.After(nowis) {
				continue
			}
			dels = append(dels, t)
			lst = append(lst, l)
		}

		for _, t := range dels {
			delete(d.d, t)
		}

		for _, l := range lst {
			for _, fun := range l {
				fun()
			}
		}
	}
}

func (d *WaitingRoom) Stop() {
	d.stopped.Store(true)
}

func (d *WaitingRoom) WaitUntil(t time.Time, fun func()) {
	common.Assert(!d.stopped.Load(), "WaitingRoom already stopped")

	d.mutex.Lock()
	defer d.mutex.Unlock()

	lst, ok := d.d[t]
	if !ok {
		lst = make([]func(), 0)
	}
	d.d[t] = append(lst, fun)
}

func (d *WaitingRoom) WaitUntilUnixSec(t uint32, fun func()) {
	d.WaitUntil(time.Unix(int64(t), 0), fun)
}

func (d *WaitingRoom) CallDelayed(t time.Duration, fun func()) {
	d.WaitUntil(time.Now().Add(t), fun)
}
