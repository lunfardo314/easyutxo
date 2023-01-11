package waitingroom

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWaitingRoom(t *testing.T) {
	wr := CreateWaitingRoom(100 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(1)

	counter := 0

	wr.WaitUntil(time.Now().Add(2*time.Second), func() {
		fmt.Printf("after 2 sec %v\n", time.Now())
		counter++
		wg.Done()
	})

	wr.CallDelayed(10*time.Millisecond, func() {
		fmt.Printf("after 10 milisec %v\n", time.Now())
		counter++
	})

	wr.CallDelayed(1*time.Second, func() {
		fmt.Printf("after 1 sec %v\n", time.Now())
		counter++
	})

	wr.CallDelayed(500*time.Millisecond, func() {
		fmt.Printf("after 500 milisec %v\n", time.Now())
		counter++
	})

	wg.Wait()
	require.EqualValues(t, 4, counter)
}
