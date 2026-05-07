package request

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/0skillallluck/scanline/utils/cacheutils"
	"github.com/0skillallluck/scanline/utils/httputils/response"
	"golang.org/x/sync/singleflight"
)

// Request represents a chainable HTTP request builder.
type Request struct {
	method         string
	url            string
	headers        http.Header
	query          url.Values
	body           io.Reader
	ctx            context.Context
	cancel         context.CancelFunc
	timeout        time.Duration
	timeoutSet     bool
	client         *http.Client
	cacheStrategy  cacheutils.Strategy
	cacheTTL       int
	cacheKeyExtras []string
	logging        bool
	redactHeaders  []string
	err            error
}

// DefaultTimeout is the default request timeout.
const DefaultTimeout = 60 * time.Second

// requestGroup deduplicates concurrent in-flight requests sharing a cache key.
var requestGroup singleflight.Group

// NewRequest creates a new Request with the given method and URL.
// By default, requests have a 60-second timeout. Use WithTimeout to override.
func NewRequest(method, rawURL string) *Request {
	return &Request{
		method:  method,
		url:     rawURL,
		headers: make(http.Header),
		query:   make(url.Values),
		ctx:     context.Background(),
		timeout: DefaultTimeout,
	}
}

// Do executes the request and returns the response.
func (r *Request) Do() (*response.Response, error) {
	// Cancel any pre-existing derived context
	if r.cancel != nil {
		defer r.cancel()
	}

	if r.err != nil {
		return nil, r.err
	}

	if r.method == http.MethodGet && r.cacheStrategy != cacheutils.None {
		return r.doCached()
	}
	return r.doFetch()
}

// doCached handles the cached path: cache check, then singleflight-wrapped
// fetch + store. Each caller unmarshals its own response from the shared
// bytes so callers can safely mutate the returned response.
func (r *Request) doCached() (*response.Response, error) {
	cacheKey := r.buildCacheKey()

	if resp, ok := r.cachedResponse(cacheKey); ok {
		return resp, nil
	}

	ch := requestGroup.DoChan(cacheKey, func() (any, error) {
		// Re-check after acquiring the slot: another goroutine may have populated it.
		// Validate before serving so corrupt entries fall through to a fresh fetch
		// instead of being passed up the stack.
		if data, found := cacheutils.Get(cacheKey, r.cacheStrategy, r.cacheTTL); found {
			if _, err := unmarshalResponse(data); err == nil {
				return data, nil
			}
			cacheutils.Delete(cacheKey)
		}
		resp, err := r.doFetch()
		if err != nil {
			return nil, err
		}
		data, err := marshalResponse(resp)
		if err != nil {
			return nil, err
		}
		if resp.IsSuccess() {
			if storeErr := cacheutils.Store(cacheKey, data, r.cacheStrategy, r.cacheTTL); storeErr != nil {
				slog.Debug("Failed to cache response",
					"error", storeErr,
					"cache_key", cacheKey,
				)
			}
		}
		return data, nil
	})

	// Honor the caller's context: a follower waiting on a slow leader's
	// fetch must be able to abandon when its own context is canceled or
	// times out. The leader's request keeps running on its own context.
	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	select {
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		if res.Shared && r.logging {
			slog.Debug("HTTP request shared via singleflight", "cache_key", cacheKey)
		}
		return unmarshalResponse(res.Val.([]byte))
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// cachedResponse returns a fresh response from cache if present and valid.
// Corrupt entries are deleted so subsequent paths can refetch.
func (r *Request) cachedResponse(cacheKey string) (*response.Response, bool) {
	data, found := cacheutils.Get(cacheKey, r.cacheStrategy, r.cacheTTL)
	if !found {
		return nil, false
	}
	resp, err := unmarshalResponse(data)
	if err != nil {
		cacheutils.Delete(cacheKey)
		return nil, false
	}
	if r.logging {
		slog.Debug("HTTP cache hit",
			"method", r.method,
			"url", r.url,
			"cache_key", cacheKey,
		)
	}
	return resp, true
}

// doFetch executes the HTTP request directly without consulting the cache.
func (r *Request) doFetch() (*response.Response, error) {
	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	reqURL, err := r.buildURL()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, r.method, reqURL, r.body)
	if err != nil {
		return nil, err
	}
	req.Header = r.headers

	start := time.Now()
	if r.logging {
		r.logRequest(req)
	}

	client := r.client
	if client == nil {
		client = DefaultClient()
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := &response.Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header,
		Body:       body,
	}

	if r.logging {
		r.logResponse(result, time.Since(start))
	}

	return result, nil
}

// DoAndDecode executes the request and decodes the JSON response into target.
// Returns an error if the response is not successful (non-2xx).
func (r *Request) DoAndDecode(target any) error {
	resp, err := r.Do()
	if err != nil {
		return err
	}

	if err := resp.CheckStatus(); err != nil {
		return err
	}

	return resp.JSON(target)
}

// buildURL constructs the final URL with query parameters.
func (r *Request) buildURL() (string, error) {
	parsed, err := url.Parse(r.url)
	if err != nil {
		return "", err
	}

	// Merge existing query params with new ones
	existingQuery := parsed.Query()
	for k, v := range r.query {
		for _, val := range v {
			existingQuery.Add(k, val)
		}
	}
	parsed.RawQuery = existingQuery.Encode()

	return parsed.String(), nil
}

// buildCacheKey generates a cache key from URL, query parameters, and any
// extras added via WithCacheKey. The whole string is later SHA256-hashed by
// cacheutils, so the format only needs to be deterministic.
func (r *Request) buildCacheKey() string {
	reqURL, _ := r.buildURL()
	if len(r.cacheKeyExtras) == 0 {
		return reqURL
	}
	parts := make([]string, 0, len(r.cacheKeyExtras)+1)
	parts = append(parts, reqURL)
	parts = append(parts, r.cacheKeyExtras...)
	return strings.Join(parts, "|")
}

// logRequest logs the outgoing request details.
func (r *Request) logRequest(req *http.Request) {
	headers := make(map[string]string)
	for k, v := range req.Header {
		if len(v) > 0 {
			if r.shouldRedact(k) {
				headers[k] = "[REDACTED]"
			} else {
				headers[k] = v[0]
			}
		}
	}

	slog.Debug("HTTP request",
		"method", req.Method,
		"url", req.URL.String(),
		"headers", headers,
	)
}

// logResponse logs the response details.
func (r *Request) logResponse(resp *response.Response, duration time.Duration) {
	slog.Debug("HTTP response",
		"status", resp.Status,
		"status_code", resp.StatusCode,
		"duration_ms", duration.Milliseconds(),
		"body_size", len(resp.Body),
	)
}

// shouldRedact checks if a header should be redacted in logs.
func (r *Request) shouldRedact(header string) bool {
	for _, h := range r.redactHeaders {
		if http.CanonicalHeaderKey(h) == http.CanonicalHeaderKey(header) {
			return true
		}
	}
	return false
}
