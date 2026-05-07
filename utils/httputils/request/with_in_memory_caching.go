package request

import "github.com/0skillallluck/scanline/utils/cacheutils"

// WithInMemoryCaching enables in-memory caching for the request.
// If ttlSeconds is 0, the entry has no explicit expiration and persists for the
// lifetime of the process (still subject to the memory cache's LRU eviction).
// If ttlSeconds > 0, the entry expires after the given number of seconds.
// Cache key is auto-calculated from URL and query parameters; use WithCacheKey
// to add additional discriminators.
// Caching only works for GET requests.
func (r *Request) WithInMemoryCaching(ttlSeconds int) *Request {
	r.cacheTTL = ttlSeconds
	r.cacheStrategy = cacheutils.MemoryOnly
	return r
}
