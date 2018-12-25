// 25.12.18 gringo
// (c) Dmitriy Blokhin [sv.dblokhin@gmail.com]. All rights reserved.
// License can be found in the LICENSE file.

package streams

import "sync"

type subStatus int

const (
	subActive subStatus = iota
	subPause
	subCancel
	subInactive
)

// Subscription concurrent safe subscription to stream
type Subscription struct {
	input      chan *streamEvent
	status     subStatus
	lock       sync.Mutex
	onNext     EventHandler
	onError    EventHandler
	onComplete EventHandler
}

func (sub *Subscription) startWorker() {
	go func() {
		for item := range sub.input {
			switch item.event {
			case streamEventData:
				sub.onNext(item.data)
			case streamEventError:
				sub.onError(item.data)
			case streamEventComplete:
				sub.onComplete(nil)
			}
		}
	}()
}

func (sub *Subscription) send(value *streamEvent) subStatus {
	sub.lock.Lock()
	defer sub.lock.Unlock()

	if sub.status == subActive {
		sub.input <- value
	}

	return sub.status
}

func (sub *Subscription) close() {
	sub.lock.Lock()
	defer sub.lock.Unlock()

	if sub.status != subInactive {
		sub.status = subInactive
		close(sub.input)
	}
}

// Cancel cancels subscription
func (sub *Subscription) Cancel() {
	sub.lock.Lock()
	defer sub.lock.Unlock()

	sub.status = subCancel
}

// Pause pauses listening stream
func (sub *Subscription) Pause() {
	sub.lock.Lock()
	defer sub.lock.Unlock()

	sub.status = subPause
}

// Resume resumes listening stream
func (sub *Subscription) Resume() {
	sub.lock.Lock()
	defer sub.lock.Unlock()

	if sub.status == subPause {
		sub.status = subActive
	}
}
