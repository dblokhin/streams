// 17.12.18 gringo
// (c) Dmitriy Blokhin [sv.dblokhin@gmail.com]. All rights reserved.
// License can be found in the LICENSE file.

package streams

import (
	"sync"
)

type S interface {
	Add(value interface{})
	Close()
	Listen(handlers ...EventHandler)
}

type EventHandler func(value interface{})

type streamHandler chan streamEvent

type streamFunc func(input chan streamEvent) chan streamEvent

// defaultStreamFunc doesn't change input data
var defaultStreamFunc streamFunc = func(input chan streamEvent) chan streamEvent {
	return input
}

const (
	// Stream statuses
	streamStatusActive int32 = iota // default value
	streamStatusClosed

	// Event types
	streamEventData eventType = iota
	streamEventError
	streamEventComplete

	maxBufferSize = 1024 * 8
)

type (
	streamEvent struct {
		event eventType
		data  interface{}
	}

	eventType int
)

type Stream struct {
	input       chan streamEvent
	fn          streamFunc
	status      int32
	statusLock  sync.Mutex
	wg          sync.WaitGroup
	listens     []streamHandler
	listensLock sync.Mutex
}

// NewStream returns created broadcast stream
func NewStream() *Stream {
	stream := &Stream{
		input:  make(chan streamEvent, maxBufferSize),
		status: streamStatusActive,
		fn:     defaultStreamFunc,
	}

	stream.wg.Add(1)
	input := stream.fn(stream.input)

	// worker
	go func() {
		for item := range input {
			stream.listensLock.Lock()
			for _, handler := range stream.listens {
				handler <- item
			}
			stream.listensLock.Unlock()
		}

		stream.listensLock.Lock()
		for _, handler := range stream.listens {
			close(handler)
		}
		stream.listensLock.Unlock()

		stream.wg.Done()
	}()

	return stream
}

// subStream creates new stream with your streamFunc that changes input data
func (s *Stream) subStream(fn streamFunc) *Stream {
	stream := &Stream{
		input:  make(chan streamEvent, maxBufferSize),
		status: streamStatusActive,
		fn:     fn,
	}

	onComplete := func(value interface{}) {
		stream.Close()
	}

	s.Listen(stream.Add, stream.AddError, onComplete)
	return stream
}

// Add adds value into stream that emits this value to listens
func (s *Stream) Add(value interface{}) {
	s.statusLock.Lock()
	if s.status == streamStatusActive {
		s.input <- streamEvent{
			event: streamEventData,
			data:  value,
		}
	}
	s.statusLock.Unlock()
}

// addArray adds array values into stream
func (s *Stream) addArray(values []interface{}) {
	for _, v := range values {
		s.Add(v)
	}
}

// AddError adds error to stream that emits this err to listens onError
func (s *Stream) AddError(value interface{}) {
	if _, ok := value.(error); !ok {
		return
	}

	s.statusLock.Lock()
	active := s.status == streamStatusActive
	s.statusLock.Unlock()

	if active {
		s.input <- streamEvent{
			event: streamEventError,
			data:  value,
		}
	}
}

// Close closes stream
func (s *Stream) Close() {
	s.statusLock.Lock()

	if s.status == streamStatusActive {
		s.status = streamStatusClosed
		s.statusLock.Unlock()

		s.input <- streamEvent{
			event: streamEventComplete,
			data:  nil,
		}
		close(s.input)
		s.wg.Wait()
		return
	}

	s.statusLock.Unlock()
}

// listen executes 3 handlers: onData, onError, OnComplete
func (s *Stream) listen(handler chan streamEvent) {
	s.listensLock.Lock()
	s.listens = append(s.listens, handler)
	s.listensLock.Unlock()
}

// Listen executes 3 handlers: onData, onError, OnComplete
func (s *Stream) Listen(handlers ...EventHandler) {
	s.statusLock.Lock()
	active := s.status == streamStatusActive
	s.statusLock.Unlock()

	if !active || len(handlers) == 0 {
		return
	}

	onNext := func(v interface{}) {
		handlers[0](v)
	}

	onError := func(v interface{}) {
		if len(handlers) > 1 {
			handlers[1](v)
		}
	}

	onComplete := func() {
		if len(handlers) > 2 {
			handlers[2](nil)
		}
	}

	eventInput := make(chan streamEvent)
	go func() {
		for item := range eventInput {
			switch item.event {
			case streamEventData:
				onNext(item.data)
			case streamEventError:
				onError(item.data)
			case streamEventComplete:
				onComplete()
			}
		}
	}()
	s.listen(eventInput)
}

// Just emits values and close stream
func (s *Stream) Just(values ...interface{}) *Stream {
	go func() {
		s.addArray(values)
		s.Close()
	}()
	return s
}
