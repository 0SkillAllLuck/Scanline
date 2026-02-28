package response

import (
	"encoding/json"
	"net/http"

	"github.com/0skillallluck/scanline/utils/httputils/errors"
)

// Response wraps an HTTP response with helper methods.
type Response struct {
	StatusCode int
	Status     string
	Headers    http.Header
	Body       []byte
}

// JSON decodes the response body as JSON into the target.
func (r *Response) JSON(target any) error {
	return json.Unmarshal(r.Body, target)
}

// String returns the response body as a string.
func (r *Response) String() string {
	return string(r.Body)
}

// Bytes returns the raw response body.
func (r *Response) Bytes() []byte {
	return r.Body
}

// IsSuccess returns true if the status code is in the 2xx range.
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// CheckStatus returns an HTTPError if the response status is not 2xx.
func (r *Response) CheckStatus() error {
	if r.IsSuccess() {
		return nil
	}
	return errors.NewHTTPError(r.StatusCode, r.Status, r.Body)
}
