package request

import "github.com/0skillallluck/scanline/utils/cacheutils"

// WithCaching enables layered caching (memory + file) for the request.
// If ttlSeconds is 0, caches indefinitely.
// If ttlSeconds > 0, caches with the specified TTL.
// Cache key is auto-calculated from URL and query parameters.
// Caching only works for GET requests.
func (r *Request) WithCaching(ttlSeconds int) *Request {
	r.cacheTTL = ttlSeconds
	r.cacheStrategy = cacheutils.Layered
	return r
}
