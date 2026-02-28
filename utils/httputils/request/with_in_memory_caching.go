package request

import "github.com/0skillallluck/scanline/utils/cacheutils"

// WithInMemoryCaching enables in-memory caching for the request.
// Cached responses persist for the lifetime of the process.
// Cache key is auto-calculated from URL and query parameters.
// Caching only works for GET requests.
func (r *Request) WithInMemoryCaching() *Request {
	r.cacheStrategy = cacheutils.MemoryOnly
	return r
}
