package g

func Lazy[T any](fn func() T) func() T {
	var result T
	var initialized bool = false
	return func() T {
		if initialized {
			return result
		}

		result = fn()
		initialized = true

		return result
	}
}
