// Package base provides the base configuration and request helpers for Plex API subpackages.
package base

import (
	"context"
	"net/http"

	"github.com/0skillallluck/scanline/utils/httputils/request"
)

// CachePolicy controls whether the Plex API helpers apply HTTP response
// caching. The two predicates are evaluated per request, so toggling the
// underlying user preference takes effect on the next call. Either field may
// be nil, which disables caching for the associated category.
type CachePolicy struct {
	// Libraries gates caching of library-listing-shaped endpoints
	// (sections, content, hubs, search, etc.).
	Libraries func() bool

	// Metadata gates caching of per-item metadata endpoints
	// (item details, children, markers, server info).
	Metadata func() bool
}

// Base contains the shared configuration for all Plex API services.
type Base struct {
	BaseURL  string
	Token    string
	ClientID string
	Cache    CachePolicy
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

// GetCachedLibraries is Get with disk + memory caching applied iff the
// Libraries cache policy is enabled. The token is folded into the cache key so
// responses don't cross between accounts pointing at the same server URL.
func (b *Base) GetCachedLibraries(ctx context.Context, path string, ttlSeconds int) *request.Request {
	req := b.Get(ctx, path)
	if b.cacheLibrariesEnabled() {
		req = req.WithCaching(ttlSeconds).WithCacheKey(b.Token)
	}
	return req
}

// GetCachedLibrariesWithQuery is GetWithQuery with disk + memory caching gated
// by the Libraries cache policy.
func (b *Base) GetCachedLibrariesWithQuery(ctx context.Context, path string, query map[string]string, ttlSeconds int) *request.Request {
	req := b.GetWithQuery(ctx, path, query)
	if b.cacheLibrariesEnabled() {
		req = req.WithCaching(ttlSeconds).WithCacheKey(b.Token)
	}
	return req
}

// GetCachedMetadata is Get with disk + memory caching gated by the Metadata
// cache policy.
func (b *Base) GetCachedMetadata(ctx context.Context, path string, ttlSeconds int) *request.Request {
	req := b.Get(ctx, path)
	if b.cacheMetadataEnabled() {
		req = req.WithCaching(ttlSeconds).WithCacheKey(b.Token)
	}
	return req
}

// GetCachedMetadataWithQuery is GetWithQuery with disk + memory caching gated
// by the Metadata cache policy.
func (b *Base) GetCachedMetadataWithQuery(ctx context.Context, path string, query map[string]string, ttlSeconds int) *request.Request {
	req := b.GetWithQuery(ctx, path, query)
	if b.cacheMetadataEnabled() {
		req = req.WithCaching(ttlSeconds).WithCacheKey(b.Token)
	}
	return req
}

// GetMemCachedLibraries is Get with in-memory-only caching gated by the
// Libraries cache policy. Use for short-lived data that shouldn't survive
// app restarts.
func (b *Base) GetMemCachedLibraries(ctx context.Context, path string, ttlSeconds int) *request.Request {
	req := b.Get(ctx, path)
	if b.cacheLibrariesEnabled() {
		req = req.WithInMemoryCaching(ttlSeconds).WithCacheKey(b.Token)
	}
	return req
}

// GetMemCachedLibrariesWithQuery is GetWithQuery with in-memory-only caching
// gated by the Libraries cache policy.
func (b *Base) GetMemCachedLibrariesWithQuery(ctx context.Context, path string, query map[string]string, ttlSeconds int) *request.Request {
	req := b.GetWithQuery(ctx, path, query)
	if b.cacheLibrariesEnabled() {
		req = req.WithInMemoryCaching(ttlSeconds).WithCacheKey(b.Token)
	}
	return req
}

// GetMemCachedMetadata is Get with in-memory-only caching gated by the
// Metadata cache policy.
func (b *Base) GetMemCachedMetadata(ctx context.Context, path string, ttlSeconds int) *request.Request {
	req := b.Get(ctx, path)
	if b.cacheMetadataEnabled() {
		req = req.WithInMemoryCaching(ttlSeconds).WithCacheKey(b.Token)
	}
	return req
}

// GetMemCachedMetadataWithQuery is GetWithQuery with in-memory-only caching
// gated by the Metadata cache policy.
func (b *Base) GetMemCachedMetadataWithQuery(ctx context.Context, path string, query map[string]string, ttlSeconds int) *request.Request {
	req := b.GetWithQuery(ctx, path, query)
	if b.cacheMetadataEnabled() {
		req = req.WithInMemoryCaching(ttlSeconds).WithCacheKey(b.Token)
	}
	return req
}

func (b *Base) cacheLibrariesEnabled() bool {
	return b.Cache.Libraries != nil && b.Cache.Libraries()
}

func (b *Base) cacheMetadataEnabled() bool {
	return b.Cache.Metadata != nil && b.Cache.Metadata()
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
