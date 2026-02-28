package signals

import (
	"sync"
)

type StatefulSignal[T any] struct {
	Signal[func(T) bool]
	currentValue T
	lock         sync.RWMutex
}

func (s *StatefulSignal[T]) CurrentValue() T {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.currentValue
}

func (s *StatefulSignal[T]) Notify(callback func(oldValue T) T) {
	s.lock.Lock()
	s.currentValue = callback(s.currentValue)
	newValue := s.currentValue
	s.lock.Unlock()

	handlers := s.Signal.snapshot()
	wg := sync.WaitGroup{}
	for sub, handler := range handlers {
		wg.Go(func() {
			if handler(newValue) {
				s.Signal.removeHandler(sub)
			}
		})
	}
	wg.Wait()
}

func (s *StatefulSignal[T]) On(handler func(T) bool) *Subscription {
	s.lock.RLock()
	defer s.lock.RUnlock()

	handler(s.currentValue)
	return s.Signal.On(handler)
}

func (s *StatefulSignal[T]) OnLazy(handler func(T) bool) *Subscription {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.Signal.On(handler)
}

func NewStatefulSignal[T any](initialValue T) *StatefulSignal[T] {
	return &StatefulSignal[T]{
		currentValue: initialValue,
		lock:         sync.RWMutex{},
		Signal:       NewSignal[func(T) bool](),
	}
}
