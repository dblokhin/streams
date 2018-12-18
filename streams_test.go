// 17.12.18 gringo
// (c) Dmitriy Blokhin [sv.dblokhin@gmail.com]. All rights reserved.
// License can be found in the LICENSE file.

package streams

import (
	"fmt"
	"sync"
	"testing"
	"time"
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
	if _, ok := <-s.input; ok {
		t.Fatal("input channel is not closed")
	}
}

func TestStream_Close2(t *testing.T) {
	s := NewStream()
	if s == nil {
		t.Fatal("failed to create new stream")
	}

	s.Close()
	s.Close()
	s.Close()
}

func TestStream_Close3(t *testing.T) {
	s := NewStream()
	if s == nil {
		t.Fatal("failed to create new stream")
	}

	const n = 100

	for i := 0; i < n; i++ {
		go s.Close()
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

	const n = 100
	wg := sync.WaitGroup{}
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(idx int) {
			s.Listen(h(idx))
			wg.Done()
		}(i)
	}

	wg.Wait()
	if len(s.listens) != n {
		t.Fatalf("invalid listens number: %d, expected: %d", len(s.listens), n)
	}
}

func TestStream_Add(t *testing.T) {
	s := NewStream()

	h := func(index int) EventHandler {
		return func(value interface{}) {
			t.Logf("from %d value: %v\n", index, value)
		}
	}

	const n = 10

	for i := 0; i < n; i++ {
		s.Listen(h(i))
	}

	for i := 0; i < 7; i++ {
		t.Logf("send value: %d", i)
		s.Add(i)
	}

	s.Close()
}

func TestStream_Add2(t *testing.T) {
	s := NewStream()

	h := func(index int) EventHandler {
		return func(value interface{}) {
			//t.Logf("from %d value: %v\n", index, value)
		}
	}

	const n = 100
	var wg1, wg2 sync.WaitGroup
	wg1.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			s.Listen(h(idx))
			wg1.Done()
		}(i)
	}

	const m = 1000
	wg2.Add(m)
	for i := 0; i < m; i++ {
		go func(idx int) {
			//t.Logf("send value: %d", i)
			s.Add(idx)
			wg2.Done()
			s.Close()
		}(i)
	}

	wg2.Wait()
	wg1.Wait()
	s.Close()
}

func TestStream_Add3(t *testing.T) {
	s := NewStream()

	h := func(index int) EventHandler {
		return func(value interface{}) {
			t.Logf("from %d value: %v\n", index, value)
		}
	}

	const n = 1

	for i := 0; i < n; i++ {
		go func(idx int) {
			s.Listen(h(idx))
		}(i)
	}

	const listensLock = 500
	s.Close()
	for i := 0; i < listensLock; i++ {
		go func(idx int) {
			//t.Logf("send value: %d", i)
			s.Add(idx)
		}(i)
	}
}

func TestStream_Just(t *testing.T) {
	s := NewStream().Just(1, 2, 3, 4)

	h := func(index int) EventHandler {
		return func(value interface{}) {
			t.Logf("from %d value: %v\n", index, value)
		}
	}

	s.Listen(h(1))

	time.Sleep(time.Millisecond * 200)
	s.Close()
}
