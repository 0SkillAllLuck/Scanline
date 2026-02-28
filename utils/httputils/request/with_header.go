package request

// WithHeader sets a single header.
func (r *Request) WithHeader(key, value string) *Request {
	r.headers.Set(key, value)
	return r
}

// WithHeaders sets multiple headers from a map.
func (r *Request) WithHeaders(headers map[string]string) *Request {
	for k, v := range headers {
		r.headers.Set(k, v)
	}
	return r
}
