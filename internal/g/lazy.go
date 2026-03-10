package g

import "sync"

func Lazy[T any](fn func() T) func() T {
	var result T
	var once sync.Once
	return func() T {
		once.Do(func() {
			result = fn()
		})
		return result
	}
}
