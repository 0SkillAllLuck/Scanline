package signals

import (
	"maps"
	"sync"

	"github.com/google/uuid"
)

const (
	Continue    = false
	Unsubscribe = true
)

type Subscription uuid.UUID

type Signal[T any] struct {
	mutex    sync.Mutex
	handlers map[*Subscription]T
}

func (b *Signal[T]) addHandler(handler T) *Subscription {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	sub := Subscription(uuid.New())
	b.handlers[&sub] = handler
	return &sub
}

func (b *Signal[T]) removeHandler(sub *Subscription) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	delete(b.handlers, sub)
}

// snapshot returns a clone of the handlers map for safe iteration.
func (b *Signal[T]) snapshot() map[*Subscription]T {
	b.mutex.Lock()
	handlers := maps.Clone(b.handlers)
	b.mutex.Unlock()
	return handlers
}

func (b *Signal[T]) On(handler T) *Subscription {
	return b.addHandler(handler)
}

func (b *Signal[T]) Unsubscribe(sub *Subscription) {
	b.removeHandler(sub)
}

func NewSignal[T any]() Signal[T] {
	return Signal[T]{
		handlers: make(map[*Subscription]T),
	}
}

func ContinueIf(condition bool) bool {
	if condition {
		return Continue
	}
	return Unsubscribe
}
