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

type Stream struct {
	input  chan interface{}
	err    chan error
	quit   chan struct{}
	update chan struct{} // update channel signals need updates cachedListeners

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
	}

	go func() {
	work:
		for {
			select {
			// update recipients
			case <-stream.update:
				stream.m.Lock()
				stream.cachedListeners = make([]streamHandler, len(stream.listeners))
				copy(stream.cachedListeners, stream.listeners)
				stream.m.Unlock()

			case item := <-stream.input:
				for _, handler := range stream.cachedListeners {
					if handler.onData != nil {
						go handler.onData(item)
					}
				}
			case err := <-stream.err:
				for _, handler := range stream.cachedListeners {
					if handler.onError != nil {
						go handler.onError(err)
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
				go handler.onCancel(nil)
			}
		}
	}()

	return stream
}

// Add adds value to stream that emits this value to listeners
func (s *Stream) Add(value interface{}) {
	s.input <- value
}

func (s *Stream) Close() {
	close(s.quit)
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
		return;
	}

	handler := streamHandler{
		onData:   onData,
		onError:  onError,
		onCancel: onCancel,
	}

	s.m.Lock()
	s.listeners = append(s.listeners, handler)
	s.m.Unlock()
}
