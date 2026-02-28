package request

import "context"

// WithContext sets the context for the request.
// This cancels any existing derived context to avoid leaks.
func (r *Request) WithContext(ctx context.Context) *Request {
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.ctx = ctx
	return r
}
