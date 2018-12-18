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
	onData     EventHandler
	onError    EventHandler
	onComplete EventHandler
}

type streamFunc func(input chan streamEvent) chan streamEvent

// defaultStreamFunc doesn't change input data
var defaultStreamFunc streamFunc = func(input chan streamEvent) chan streamEvent {
	return input
}

type streamStatus int

const (
	// Stream statuses
	streamActive streamStatus = iota // default value
	streamClosed

	// Event types
	streamData  eventType = iota
	streamError
	streamComplete

	maxBufferSize = 1024*8
)

type (
	streamEvent struct {
		event eventType
		data  interface{}
	}

	eventType int
)

type Stream struct {
	input  chan streamEvent
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
		input:  make(chan streamEvent, maxBufferSize),
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
				switch item.event {
				case streamData:
					for _, handler := range stream.cachedListeners {
						if handler.onData != nil {
							handler.onData(item.data)
						}
					}

				case streamError:
					for _, handler := range stream.cachedListeners {
						if handler.onError != nil {
							handler.onError(item.data)
						}
					}

				case streamComplete:
					for _, handler := range stream.cachedListeners {
						if handler.onComplete != nil {
							handler.onComplete(nil)
						}
					}

					break work;
				}
			}
		}

		close(stream.input)
		close(stream.update)
		stream.wg.Done()
	}()

	return stream
}

// subStream creates new stream with your streamFunc that changes input data
func (s *Stream) subStream(fn streamFunc) *Stream {
	stream := &Stream{
		input:  make(chan streamEvent, maxBufferSize),
		update: make(chan struct{}),
		fn:     fn,
	}

	onComplete := func(value interface{}) {
		stream.Close()
	}

	s.Listen(stream.Add, stream.AddError, onComplete)
	return stream
}

// Add adds value into stream that emits this value to listeners
func (s *Stream) Add(value interface{}) {
	if s.status == streamActive {
		s.input <- streamEvent{
			event: streamData,
			data:  value,
		}
	}
}

// AddArray adds array values into stream
func (s *Stream) AddArray(values []interface{}) {
	for _, v := range values {
		s.Add(v)
	}
}

// AddError adds error to stream that emits this err to listeners onError
func (s *Stream) AddError(value interface{}) {
	if s.status == streamActive {
		s.input <- streamEvent{
			event: streamError,
			data:  value,
		}
	}
}

// Close closes stream
func (s *Stream) Close() {
	if s.status == streamActive {
		s.status = streamClosed

		s.input <- streamEvent{
			event: streamComplete,
			data:  nil,
		}
		s.wg.Wait()
	}
}

// Listen executes 3 handlers: onData, onError, OnComplete
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
		onData:     onData,
		onError:    onError,
		onComplete: onCancel,
	}

	s.m.Lock()
	s.listeners = append(s.listeners, handler)
	s.m.Unlock()
	s.update <- struct{}{}
}

// Just emits values and close stream
func (s *Stream) Just(value interface{}, values ...interface{}) *Stream {
	go func() {
		s.AddArray(values)
		s.Close()
	}()
	return s
}
