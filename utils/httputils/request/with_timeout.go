package request

import (
	"time"
)

// WithTimeout sets a timeout for the request.
// The timeout is applied lazily in Do() as a child of the base context.
func (r *Request) WithTimeout(timeout time.Duration) *Request {
	r.timeout = timeout
	r.timeoutSet = true
	return r
}
