package request

import (
	"bytes"
	"encoding/json"
	"io"
)

// WithJSONBody sets the request body as JSON.
func (r *Request) WithJSONBody(body any) *Request {
	data, err := json.Marshal(body)
	if err != nil {
		r.err = err
		return r
	}
	r.body = bytes.NewReader(data)
	r.headers.Set("Content-Type", "application/json")
	return r
}

// WithBody sets a raw request body.
func (r *Request) WithBody(body io.Reader) *Request {
	r.body = body
	return r
}
