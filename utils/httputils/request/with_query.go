package request

import "net/url"

// WithQuery sets URL query parameters.
func (r *Request) WithQuery(params map[string]string) *Request {
	for k, v := range params {
		r.query.Set(k, v)
	}
	return r
}

// WithQueryValues sets URL query parameters from url.Values.
// This preserves multiple values for the same key.
func (r *Request) WithQueryValues(values url.Values) *Request {
	for k, vals := range values {
		for _, v := range vals {
			r.query.Add(k, v)
		}
	}
	return r
}
