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
	s.WaitDone()
	s.WaitDone()
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

func TestStream_Listen2(t *testing.T) {
	s := NewStream()
	defer s.Close()

	h := func(index int) EventHandler {
		return func(value interface{}) {
			fmt.Printf("from %d value: %v", index, value)
		}
	}

	const n = 100

	for i := 0; i < n; i++ {
		s = s.Listen(h(i)).Listen(h(i + 1000)).Listen(h(i + 1000000))
	}

	if len(s.listens) != n*3 {
		t.Fatalf("invalid listens number: %d, expected: %d", len(s.listens), n*3)
	}
}

func TestStream_Add(t *testing.T) {
	const (
		n = 10
		m = 10
	)

	ans := 0
	lock := sync.Mutex{}

	handler := func(v interface{}) {
		lock.Lock()
		ans += v.(int)
		lock.Unlock()
	}

	s := NewStream()
	for i := 0; i < n; i++ {
		s.Listen(handler)
	}

	for i := 1; i <= m; i++ {
		s.Add(i)
	}

	s.Close()

	expect := ((m * (m + 1)) / 2) * n

	if ans != expect {
		t.Fatalf("invalid answer: %d, expected: %d", ans, expect)
	}
}

func TestStream_Add2(t *testing.T) {
	const (
		n = 10
		m = 100
	)

	ans := 0
	lock := sync.Mutex{}

	handler := func(v interface{}) {
		lock.Lock()
		ans += v.(int)
		lock.Unlock()
	}

	var wg sync.WaitGroup
	s := NewStream()

	// add listeners
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			s.Listen(handler)
			wg.Done()
		}()
	}
	wg.Wait()

	// add values
	wg.Add(m)
	for i := 1; i <= m; i++ {
		go func(idx int) {
			s.Add(idx)
			wg.Done()
		}(i)
	}

	wg.Wait()
	s.Close()

	expect := ((m * (m + 1)) / 2) * n
	if ans != expect {
		t.Fatalf("invalid answer: %d, expected: %d", ans, expect)
	}
}

func TestStream_Just(t *testing.T) {
	ans := 0
	var m sync.Mutex
	handler := func(value interface{}) {
		m.Lock()
		ans++
		ans += value.(int)
		m.Unlock()
	}

	Just(1, 2, 3, 4).Listen(handler).WaitDone()

	if ans != 14 {
		t.Fatalf("invalid answer: %d, expected: %d", ans, 14)
	}
}

func TestStream_Filter(t *testing.T) {
	ans := 0
	var m sync.Mutex

	h := func(value interface{}) {
		v := value.(int)
		m.Lock()
		ans += v
		m.Unlock()
	}

	filter := func(x interface{}) bool {
		return x.(int) > 3
	}

	Just(1, 2, 3, 4, 10, 100, 1000, 0, 1).Filter(filter).Listen(h).WaitDone()

	if ans != 1114 {
		t.Fatalf("invalid answer: %d, expected: %d", ans, 1114)
	}
}

func TestStream_First(t *testing.T) {

	ans := 0
	var m sync.Mutex
	handler := func(value interface{}) {
		m.Lock()
		ans = value.(int)
		m.Unlock()
	}

	filter := func(value interface{}) bool {
		return value.(int) > 10
	}

	Just(1, 2, 42, 3).Filter(filter).First().Listen(handler).WaitDone()
	if ans != 42 {
		t.Fatalf("invalid answer: %d, expected: %d", ans, 42)
	}
}

func TestStream_Last(t *testing.T) {
	ans := 0
	var m sync.Mutex
	handler := func(value interface{}) {
		m.Lock()
		ans = value.(int)
		m.Unlock()
	}

	filter := func(value interface{}) bool {
		return value.(int) > 10
	}

	Just(1, 2, 42, 3).Filter(filter).Last().Listen(handler).WaitDone()
	if ans != 42 {
		t.Fatalf("invalid answer: %d, expected: %d", ans, 42)
	}
}

func TestStream_WaitDone(t *testing.T) {
	// check for empty stream
	st1 := Just().WaitDone()
	if st1.status != streamStatusClosed {
		t.Fatalf("invalid status of closed channel: %d", st1.status)
	}

	st2 := Just(1, 2, 3).WaitDone()
	if st2.status != streamStatusClosed {
		t.Fatalf("invalid status of closed channel: %d", st2.status)
	}
}

func BenchmarkStream_Add_1Handler(b *testing.B) {
	handler := func(value interface{}) {
	}

	s := NewStream()
	s.Listen(handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Add(i)
	}

	s.Close()
	s.WaitDone()
}

func BenchmarkStream_Add_NHandlers(b *testing.B) {
	s := NewStream()

	h := func(index int) EventHandler {
		return func(value interface{}) {
		}
	}

	const n = 100

	for i := 0; i < n; i++ {
		s.Listen(h(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Add(i)
	}

	s.Close()
	s.WaitDone()
}
