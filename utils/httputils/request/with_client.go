package request

import "net/http"

// WithClient sets a custom HTTP client for the request.
func (r *Request) WithClient(client *http.Client) *Request {
	r.client = client
	return r
}
