// 17.12.18 gringo
// (c) Dmitriy Blokhin [sv.dblokhin@gmail.com]. All rights reserved.
// License can be found in the LICENSE file.

package streams

import (
	"testing"
	"time"
	"fmt"
	"sync"
)

func TestNewStream(t *testing.T) {
	s := NewStream()
	if s == nil {
		t.Fatal("failed to create new stream")
	}

	s.Close()
}

func TestStream_Close(t *testing.T) {
	s := NewStream()
	if s == nil {
		t.Fatal("failed to create new stream")
	}

	s.Close()

	time.Sleep(time.Millisecond * 200)

	// checks for closed channels
	if _, ok := <-s.quit; ok {
		t.Fatal("quit channel is not closed")
	}

	if _, ok := <-s.input; ok {
		t.Fatal("input channel is not closed")
	}

	if _, ok := <-s.update; ok {
		t.Fatal("update channel is not closed")
	}

	if _, ok := <-s.err; ok {
		t.Fatal("err channel is not closed")
	}
}

func TestStream_Listen(t *testing.T) {
	s := NewStream()
	defer s.Close()

	h := func(index int) EventHandler {
		return func(value interface{}) {
			fmt.Printf("from %d value: %v", index, value)
		}
	}

	const n = 100;
	wg := sync.WaitGroup{}
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(idx int) {
			s.Listen(h(idx))
			wg.Done()
		}(i)
	}

	wg.Wait()
	if len(s.cachedListeners) != n {
		t.Fatalf("invalid listeners number: %d, expected: %d", len(s.cachedListeners), n)
	}
}

func TestStream_Add(t *testing.T) {
	s := NewStream()
	defer s.Close()

	h := func(index int) EventHandler {
		return func(value interface{}) {
			t.Logf("from %d value: %v\n", index, value)
		}
	}

	const n = 10
	wg := sync.WaitGroup{}
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(idx int) {
			s.Listen(h(idx))
			wg.Done()
		}(i)
	}

	wg.Wait()

	for i := 0; i < 17; i++ {
		s.Add(i)
	}

	time.Sleep(time.Millisecond * 200)
}
