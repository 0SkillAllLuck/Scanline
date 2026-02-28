// Package base provides the base configuration and request helpers for Plex API subpackages.
package base

import (
	"context"
	"net/http"

	"github.com/0skillallluck/scanline/utils/httputils/request"
)

// Base contains the shared configuration for all Plex API services.
type Base struct {
	BaseURL  string
	Token    string
	ClientID string
}

// Request builds a new request with Plex headers pre-configured.
func (b *Base) Request(method, path string) *request.Request {
	return request.NewRequest(method, b.BaseURL+path).
		WithHeaders(map[string]string{
			"X-Plex-Token":             b.Token,
			"X-Plex-Client-Identifier": b.ClientID,
			"Accept":                   "application/json",
		}).
		WithLogging("X-Plex-Token")
}

// Get returns a GET request ready for execution.
func (b *Base) Get(ctx context.Context, path string) *request.Request {
	return b.Request(http.MethodGet, path).WithContext(ctx)
}

// GetWithQuery returns a GET request with query parameters.
func (b *Base) GetWithQuery(ctx context.Context, path string, query map[string]string) *request.Request {
	return b.Request(http.MethodGet, path).
		WithContext(ctx).
		WithQuery(query)
}

// Post returns a POST request.
func (b *Base) Post(ctx context.Context, path string) *request.Request {
	return b.Request(http.MethodPost, path).WithContext(ctx)
}

// PostWithQuery returns a POST request with query parameters.
func (b *Base) PostWithQuery(ctx context.Context, path string, query map[string]string) *request.Request {
	return b.Request(http.MethodPost, path).
		WithContext(ctx).
		WithQuery(query)
}

// Put returns a PUT request.
func (b *Base) Put(ctx context.Context, path string) *request.Request {
	return b.Request(http.MethodPut, path).WithContext(ctx)
}

// PutWithQuery returns a PUT request with query parameters.
func (b *Base) PutWithQuery(ctx context.Context, path string, query map[string]string) *request.Request {
	return b.Request(http.MethodPut, path).
		WithContext(ctx).
		WithQuery(query)
}

// Delete returns a DELETE request.
func (b *Base) Delete(ctx context.Context, path string) *request.Request {
	return b.Request(http.MethodDelete, path).WithContext(ctx)
}
