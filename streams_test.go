// 17.12.18 gringo
// (c) Dmitriy Blokhin [sv.dblokhin@gmail.com]. All rights reserved.
// License can be found in the LICENSE file.

package streams

import (
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
	if _, ok := <- s.quit; ok {
		t.Fatal("quit channel is not closed")
	}

	if _, ok := <- s.input; ok {
		t.Fatal("input channel is not closed")
	}

	if _, ok := <- s.update; ok {
		t.Fatal("update channel is not closed")
	}

	if _, ok := <- s.err; ok {
		t.Fatal("err channel is not closed")
	}
}

