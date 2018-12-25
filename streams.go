// 17.12.18 gringo
// (c) Dmitriy Blokhin [sv.dblokhin@gmail.com]. All rights reserved.
// License can be found in the LICENSE file.

package streams

import (
	"sync"
	"time"
)

type S interface {
	Add(value interface{})
	Close()
	Listen(handlers ...EventHandler)
}

type EventHandler func(value interface{})

type streamHandler chan *streamEvent

// streamFunc changes input event and emmit it to new channel.
// new channel MUST be closed when input closed
type streamFunc func(s *Stream, input chan *streamEvent) chan *streamEvent

// defaultStreamFunc doesn't change input data
var defaultStreamFunc streamFunc = func(s *Stream, input chan *streamEvent) chan *streamEvent {
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

	defaultBufferSize = 50
)

type (
	streamEvent struct {
		event eventType
		data  interface{}
	}

	eventType int
)

type Stream struct {
	input      chan *streamEvent
	bufferSize int
	fn         streamFunc
	status     int32
	listens    []Subscription
	lock       sync.Mutex
	wg         sync.WaitGroup
}

var completeStreamEvent = &streamEvent{
	event: streamEventComplete,
	data:  nil,
}

// NewStream returns created broadcast stream
func NewStream() *Stream {
	stream := &Stream{
		input:      make(chan *streamEvent, defaultBufferSize),
		bufferSize: defaultBufferSize,
		status:     streamStatusActive,
		fn:         defaultStreamFunc,
	}

	stream.startWorker()
	return stream
}

func NewStreamSize(bufferSize int) *Stream {
	stream := &Stream{
		input:      make(chan *streamEvent, bufferSize),
		bufferSize: bufferSize,
		status:     streamStatusActive,
		fn:         defaultStreamFunc,
	}

	stream.startWorker()
	return stream
}

func (s *Stream) startWorker() {
	s.wg.Add(1)
	input := s.fn(s, s.input)

	go func() {
		for item := range input {
			s.propagateItem(item)
		}

		s.closeListens()
		s.wg.Done()
	}()
}

func (s *Stream) propagateItem(item *streamEvent) {
	for idx, sub := range s.listens {
		// remove canceled subscription
		if sub.send(item) == subCancel {
			go func(remove int) {
				s.lock.Lock()
				s.listens[remove].close()
				s.listens = append(s.listens[:remove], s.listens[remove+1:]...)
				s.lock.Unlock()
			}(idx)
		}
	}
}

func (s *Stream) closeListens() {
	for _, sub := range s.listens {
		sub.close()
	}
}

func (s *Stream) closeWorker() {
	s.input <- completeStreamEvent
	close(s.input)
}

func (s *Stream) isActive() bool {
	return s.status == streamStatusActive
}

// Close closes stream
func (s *Stream) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isActive() {
		s.status = streamStatusClosed
		s.closeWorker()
		s.wg.Wait()
	}
}

// subStream creates new stream with your streamFunc that changes input data
func (s *Stream) subStream(fn streamFunc) *Stream {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isActive() {
		return s
	}

	stream := &Stream{
		input:      make(chan *streamEvent, s.bufferSize),
		bufferSize: s.bufferSize,
		status:     streamStatusActive,
		fn:         fn,
	}
	stream.startWorker()

	sub := Subscription{
		input:   make(chan *streamEvent),
		onNext:  stream.Add,
		onError: stream.AddError,
		onComplete: func(value interface{}) {
			stream.Close()
		},
	}
	sub.startWorker()
	s.listens = append(s.listens, sub)

	return stream
}

// Add adds value into stream that emits this value to listens
func (s *Stream) Add(value interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isActive() {
		return
	}

	s.input <- &streamEvent{
		event: streamEventData,
		data:  value,
	}
}

// addArray adds array values into stream
func (s *Stream) addArray(values []interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isActive() {
		return
	}

	for _, v := range values {
		s.input <- &streamEvent{
			event: streamEventData,
			data:  v,
		}
	}
}

// AddError adds error to stream that emits this err to listens onError
func (s *Stream) AddError(value interface{}) {
	if _, ok := value.(error); !ok {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isActive() {
		return
	}

	s.input <- &streamEvent{
		event: streamEventError,
		data:  value,
	}
}

// Listen executes 3 handlers: onData, onError, OnComplete
func (s *Stream) Listen(handlers ...EventHandler) *Stream {

	if len(handlers) == 0 {
		return s
	}

	onNext := func(v interface{}) {
		handlers[0](v)
	}

	onError := func(v interface{}) {
		if len(handlers) > 1 {
			handlers[1](v)
		}
	}

	onComplete := func(v interface{}) {
		if len(handlers) > 2 {
			handlers[2](nil)
		}
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isActive() {
		return s
	}

	sub := Subscription{
		status:     subActive,
		input:      make(chan *streamEvent),
		onNext:     onNext,
		onError:    onError,
		onComplete: onComplete,
	}

	sub.startWorker()
	s.listens = append(s.listens, sub)

	return s
}

// WaitDone waits until stream is closed
func (s *Stream) WaitDone() *Stream {
	s.wg.Wait()
	return s
}

// Filter filters elements emitted from streams
func (s *Stream) Filter(apply func(value interface{}) bool) *Stream {
	fn := func(s *Stream, input chan *streamEvent) chan *streamEvent {
		output := make(chan *streamEvent)
		go func() {
			for item := range input {
				if item.event == streamEventData {
					if apply(item.data) {
						output <- item
					}
					continue
				}

				// pass Error & Complete events always
				output <- item
			}

			close(output)
		}()

		return output
	}

	return s.subStream(fn)
}

// Map
func (s *Stream) Map(apply func(value interface{}) interface{}) *Stream {
	fn := func(s *Stream, input chan *streamEvent) chan *streamEvent {
		output := make(chan *streamEvent)
		go func() {
			for item := range input {
				// pass Error & Complete events always
				if item.event == streamEventError || item.event == streamEventComplete {
					output <- item
					continue
				}

				item.data = apply(item.data)
				output <- item
			}

			close(output)
		}()

		return output
	}

	return s.subStream(fn)
}

// Take
func (s *Stream) Take(n int) *Stream {
	fn := func(s *Stream, input chan *streamEvent) chan *streamEvent {
		output := make(chan *streamEvent)
		go func() {
			if n <= 0 {
				return
			}

			for item := range input {
				output <- item

				if item.event == streamEventData {
					n--
					if n <= 0 {
						break
					}
				}
			}

			// if original stream is not closed
			if n <= 0 {
				output <- completeStreamEvent
				go s.Close()
			}

			close(output) // close stream worker
		}()

		return output
	}

	return s.subStream(fn)
}

// TakeLast
func (s *Stream) TakeLast(n int) *Stream {
	fn := func(s *Stream, input chan *streamEvent) chan *streamEvent {
		output := make(chan *streamEvent)
		go func() {
			buf := make([]*streamEvent, n)
			for item := range input {
				if item.event == streamEventData {
					if len(buf) >= n {
						buf = buf[1:]
					}
					buf = append(buf, item)
					continue
				}

				// pass Error & Complete events always
				output <- item
			}

			for _, v := range buf {
				output <- v
			}

			close(output) // close stream worker
		}()

		return output
	}

	return s.subStream(fn)
}

// First returns stream that emmit first emitted value of the underlying stream
func (s *Stream) First() *Stream {
	fn := func(s *Stream, input chan *streamEvent) chan *streamEvent {
		output := make(chan *streamEvent)
		go func() {
			closed := true
			for item := range input {
				output <- item

				if item.event == streamEventData {
					closed = false
					break
				}
			}

			if !closed {
				output <- completeStreamEvent
				go s.Close()
			}

			close(output) // close stream worker
		}()

		return output
	}

	return s.subStream(fn)
}

// Last returns stream that emmit last emitted value of the underlying stream
func (s *Stream) Last() *Stream {
	fn := func(s *Stream, input chan *streamEvent) chan *streamEvent {
		output := make(chan *streamEvent)
		go func() {
			lastItem := completeStreamEvent
			for item := range input {
				if item.event == streamEventData {
					lastItem = item
					continue
				}

				output <- item
			}

			if lastItem.event == streamEventData {
				output <- lastItem
			}

			close(output)
		}()

		return output
	}

	return s.subStream(fn)
}

// Distinct
func (s *Stream) Distinct(apply func(value interface{}) interface{}) *Stream {
	fn := func(s *Stream, input chan *streamEvent) chan *streamEvent {
		output := make(chan *streamEvent)
		go func() {
			dups := make(map[interface{}]*struct{})
			for item := range input {
				if item.event == streamEventData {
					if _, exist := dups[item.data]; !exist {
						output <- item
						dups[item.data] = nil
					}
					continue
				}

				// pass Error & Complete events always
				output <- item
			}

			close(output)
		}()

		return output
	}

	return s.subStream(fn)
}

// Debounce emits the latest value in given time window
func (s *Stream) Debounce(d time.Duration) *Stream {
	fn := func(s *Stream, input chan *streamEvent) chan *streamEvent {
		output := make(chan *streamEvent)
		go func() {
			tick := time.NewTicker(d)
			lastValue := <-input

			for lastValue != nil {
				select {
				case <-tick.C:
					output <- lastValue
				case item := <-input:
					if item.event == streamEventData {
						lastValue = item
						continue
					}

					// pass Error & Complete events always
					output <- item
				}
			}

			tick.Stop()
			close(output)
		}()

		return output
	}

	return s.subStream(fn)
}

// ElementAt
func (s *Stream) ElementAt(n int) *Stream {
	return s.Skip(n - 1).First()
}

// Skip
func (s *Stream) Skip(n int) *Stream {
	fn := func(s *Stream, input chan *streamEvent) chan *streamEvent {
		output := make(chan *streamEvent)
		go func() {
			for item := range input {
				if n <= 0 {
					output <- item
					continue
				}

				if item.event == streamEventData {
					n--
					continue
				}

				output <- item
			}

			close(output) // close stream worker
		}()

		return output
	}

	return s.subStream(fn)
}

// SkipLast
func (s *Stream) SkipLast(n int) *Stream {
	fn := func(s *Stream, input chan *streamEvent) chan *streamEvent {
		output := make(chan *streamEvent)
		go func() {
			buf := make(chan *streamEvent, n)
			for item := range input {
				// pass Error & Complete events always
				if item.event == streamEventError || item.event == streamEventComplete {
					output <- item
					continue
				}

				select {
				case buf <- item:
				default:
					output <- <-buf
					buf <- item
				}
			}

			close(buf)
			close(output)
		}()

		return output
	}

	return s.subStream(fn)
}

// Just emits values and close stream
func Just(values ...interface{}) *Stream {
	stream := NewStream()
	go func() {
		stream.addArray(values)
		stream.Close()
	}()
	return stream
}
