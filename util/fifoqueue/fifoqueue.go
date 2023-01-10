package fifoqueue

import (
	"sync"

	"github.com/gammazero/deque"
)

// FIFOQueue implements variable size synchronized FIFO queue
type FIFOQueue[T any] struct {
	d       *deque.Deque[T]
	mutex   sync.Mutex
	out     chan T
	closing bool
	once    sync.Once
}

func NewFIFOQueue[T any]() *FIFOQueue[T] {
	return &FIFOQueue[T]{
		d:   new(deque.Deque[T]),
		out: make(chan T),
	}
}

// Write pushes element
func (q *FIFOQueue[T]) Write(elem T) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closing {
		panic("attempt to write to the closed FIFOQueue")
	}

	q.d.PushBack(elem)

	e := q.d.Front()
	select {
	case q.out <- e:
		q.d.PopFront()
	default:
	}
}

// CloseNow closes FIFOQueue immediately. The elements in the buffer are lost
func (q *FIFOQueue[T]) CloseNow() {
	close(q.out)
}

// Close closes FIFOQueue deferred until all elements are read
func (q *FIFOQueue[T]) Close() {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.closing = true
	if q.d.Len() == 0 {
		q.once.Do(func() {
			close(q.out)
		})
	}
}

func (q *FIFOQueue[T]) read() (T, bool) {
	select {
	case ret, ok := <-q.out:
		return ret, ok
	default:
	}

	q.mutex.Lock()

	if q.d.Len() > 0 {
		defer q.mutex.Unlock()

		return q.d.PopFront(), true
	}
	// empty buffer
	if q.closing {
		q.once.Do(func() {
			close(q.out)
		})
	}
	q.mutex.Unlock()

	ret, ok := <-q.out
	return ret, ok
}

// Consume reads all elements of the queue until it is closed
func (q *FIFOQueue[T]) Consume(fun func(elem T)) {
	for {
		e, ok := q.read()
		if !ok {
			break
		}
		fun(e)
	}
}

// Len returns number of elements in the queue. Non-deterministic
func (q *FIFOQueue[T]) Len() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	return q.d.Len()
}
