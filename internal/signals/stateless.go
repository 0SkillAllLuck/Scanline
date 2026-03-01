package signals

import "sync"

type StatelessSignal[T any] struct {
	Signal[func(T) bool]
}

func (s *StatelessSignal[T]) Notify(newValue T) {
	handlers := s.snapshot()
	wg := sync.WaitGroup{}
	for sub, handler := range handlers {
		wg.Go(func() {
			if handler(newValue) {
				s.removeHandler(sub)
			}
		})
	}
	wg.Wait()
}

func (s *StatelessSignal[T]) On(handler func(T) bool) *Subscription {
	return s.Signal.On(handler)
}

func NewStatelessSignal[T any]() *StatelessSignal[T] {
	return &StatelessSignal[T]{
		Signal: NewSignal[func(T) bool](),
	}
}
