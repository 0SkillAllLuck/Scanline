package request

// WithCacheKey appends extra strings to the cache key for this request.
// Useful for discriminating cached responses by something not present in the
// URL (e.g. an auth token), so that two callers with the same URL but
// different identities don't share cache entries.
// The extras are joined into the cache key with a separator and the whole
// string is SHA256-hashed by cacheutils, so any value is safe to pass.
func (r *Request) WithCacheKey(extras ...string) *Request {
	r.cacheKeyExtras = append(r.cacheKeyExtras, extras...)
	return r
}
