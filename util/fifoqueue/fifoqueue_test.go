package fifoqueue

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		q := New[string]()
		require.EqualValues(t, 0, q.Len())
		q.Write("one")
		require.EqualValues(t, 1, q.Len())
		q.Write("two")
		require.EqualValues(t, 2, q.Len())
		e, ok := q.read()
		require.True(t, ok)
		require.EqualValues(t, "one", e)
		require.EqualValues(t, 1, q.Len())
		e, ok = q.read()
		require.True(t, ok)
		require.EqualValues(t, "two", e)
		require.EqualValues(t, 0, q.Len())
	})
	t.Run("2", func(t *testing.T) {
		q := New[string]()
		require.EqualValues(t, 0, q.Len())
		q.Write("one")
		require.EqualValues(t, 1, q.Len())

		q.CloseNow()
		_, ok := q.read()
		require.False(t, ok)
		require.EqualValues(t, 1, q.Len())
	})
	t.Run("4", func(t *testing.T) {
		q := New[int]()
		require.EqualValues(t, 0, q.Len())
		for i := 0; i < 10000; i++ {
			q.Write(i)
			require.EqualValues(t, i+1, q.Len())
		}
		for i := 0; i < 10000; i++ {
			ib, ok := q.read()
			require.True(t, ok)
			require.EqualValues(t, i, ib)
		}
		require.EqualValues(t, 0, q.Len())
	})
}

func TestMultiThread1(t *testing.T) {
	q := New[int]()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for i := 0; i < 100; i++ {
			q.Write(i)
		}
		q.Close()
	}()
	count := 0
	go func() {
		for i := 0; i < 10000; i++ {
			ib, ok := q.read()
			if !ok {
				t.Logf("stop at count %d", count)
				break
			}
			require.True(t, ok)
			require.EqualValues(t, i, ib)
			count++
			time.Sleep(1 * time.Millisecond)
		}
		wg.Done()
	}()
	wg.Wait()
	require.EqualValues(t, 100, count)
	require.EqualValues(t, 0, q.Len())
}

func TestMultiThread2(t *testing.T) {
	q := New[int]()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for i := 0; i < 100; i++ {
			q.Write(i)
		}
		q.Close()
	}()
	count := 0
	go func() {
		q.Consume(func(e int) {
			//t.Logf("%d", e)
			count++
			time.Sleep(1 * time.Millisecond)
		})
		wg.Done()
	}()
	wg.Wait()
	require.EqualValues(t, 100, count)
	require.EqualValues(t, 0, q.Len())
}

func BenchmarkRW(b *testing.B) {
	q := New[int]()
	for i := 0; i < b.N; i++ {
		q.Write(i)
	}
	for i := 0; i < b.N; i++ {
		_, _ = q.read()
	}
}
