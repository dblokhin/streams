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

type streamHandler struct {
	onData   EventHandler
	onError  EventHandler
	onCancel EventHandler
}

type streamFunc func(input chan interface{}) chan interface{}

// defaultStreamFunc doesn't change input data
var defaultStreamFunc streamFunc = func(input chan interface{}) chan interface{} {
	return input
}

type streamStatus int

const (
	streamActive streamStatus = iota // default value
	streamClosed
)

type Stream struct {
	input  chan interface{}
	err    chan error
	quit   chan struct{}
	update chan struct{} // update channel signals need updates cachedListeners
	fn     streamFunc
	status streamStatus

	wg              sync.WaitGroup
	cachedListeners []streamHandler
	listeners       []streamHandler
	m               sync.Mutex
}

// NewStream returns created broadcast stream
func NewStream() *Stream {
	stream := &Stream{
		input:  make(chan interface{}, 100),
		err:    make(chan error),
		quit:   make(chan struct{}),
		update: make(chan struct{}),
		fn:     defaultStreamFunc,
	}

	stream.wg.Add(1)
	input := stream.fn(stream.input)

	go func() {
	work:
		for {
			select {
			// update cached recipients
			case <-stream.update:
				stream.m.Lock()
				stream.cachedListeners = make([]streamHandler, len(stream.listeners))
				copy(stream.cachedListeners, stream.listeners)
				stream.m.Unlock()

			case item := <-input:
				for _, handler := range stream.cachedListeners {
					if handler.onData != nil {
						handler.onData(item)
					}
				}
			case err := <-stream.err:
				for _, handler := range stream.cachedListeners {
					if handler.onError != nil {
						handler.onError(err)
					}
				}
			case <-stream.quit:
				break work
			}
		}

		close(stream.input)
		close(stream.err)
		close(stream.update)

		for _, handler := range stream.cachedListeners {
			if handler.onCancel != nil {
				handler.onCancel(nil)
			}
		}

		stream.wg.Done()
	}()

	return stream
}

// subStream creates new stream with your streamFunc that changes input data
func (s *Stream) subStream(fn streamFunc) *Stream {
	stream := &Stream{
		input:  make(chan interface{}, 100),
		err:    make(chan error),
		quit:   make(chan struct{}),
		update: make(chan struct{}),
		fn:     fn,
	}

	onCancel := func(value interface{}) {
		stream.Close()
	}

	s.Listen(stream.Add, stream.Error, onCancel)
	return stream
}

// Add adds value into stream that emits this value to listeners
func (s *Stream) Add(value interface{}) {
	if s.status == streamActive {
		s.input <- value
	}
}

// AddArray adds array values into stream
func (s *Stream) AddArray(values []interface{}) {
	for _, v := range values {
		s.Add(v)
	}
}

// Error adds error to stream that emits this err to listeners onError
func (s *Stream) Error(value interface{}) {
	if s.status == streamActive {
		if err, ok := value.(error); ok {
			s.err <- err
		}
	}
}

// Close closes stream
func (s *Stream) Close() {
	s.status = streamClosed
	close(s.quit)
	s.wg.Wait()
}

// Listen executes 3 handlers: onData, onError, OnCancel
func (s *Stream) Listen(handlers ...EventHandler) {
	if len(handlers) == 0 {
		return
	}

	var onData, onError, onCancel EventHandler

	switch len(handlers) {
	case 3:
		onCancel = handlers[2]
		fallthrough
	case 2:
		onError = handlers[1]
		fallthrough
	case 1:
		onData = handlers[0]

	default:
		return
	}

	handler := streamHandler{
		onData:   onData,
		onError:  onError,
		onCancel: onCancel,
	}

	s.m.Lock()
	s.listeners = append(s.listeners, handler)
	s.m.Unlock()
	s.update <- struct{}{}
}

// Just emits values and close stream
func (s *Stream) Just(values []interface{}) {
	s.AddArray(values)
	s.Close()
}
