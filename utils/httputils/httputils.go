package httputils

import (
	"net/http"

	"github.com/0skillallluck/scanline/utils/httputils/request"
)

// Get creates a new GET request.
func Get(url string) *request.Request {
	return request.NewRequest(http.MethodGet, url)
}

// Post creates a new POST request.
func Post(url string) *request.Request {
	return request.NewRequest(http.MethodPost, url)
}

// Put creates a new PUT request.
func Put(url string) *request.Request {
	return request.NewRequest(http.MethodPut, url)
}

// Patch creates a new PATCH request.
func Patch(url string) *request.Request {
	return request.NewRequest(http.MethodPatch, url)
}

// Delete creates a new DELETE request.
func Delete(url string) *request.Request {
	return request.NewRequest(http.MethodDelete, url)
}
