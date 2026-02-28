package request

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/0skillallluck/scanline/utils/cacheutils"
	"github.com/0skillallluck/scanline/utils/httputils/response"
)

// Request represents a chainable HTTP request builder.
type Request struct {
	method        string
	url           string
	headers       http.Header
	query         url.Values
	body          io.Reader
	ctx           context.Context
	cancel        context.CancelFunc
	timeout       time.Duration
	timeoutSet    bool
	client        *http.Client
	cacheStrategy cacheutils.Strategy
	cacheTTL      int
	logging       bool
	redactHeaders []string
	err           error
}

// DefaultTimeout is the default request timeout.
const DefaultTimeout = 60 * time.Second

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

	// Check cache for GET requests
	if r.method == http.MethodGet && r.cacheStrategy != cacheutils.None {
		cacheKey := r.buildCacheKey()
		if data, found := cacheutils.Get(cacheKey, r.cacheStrategy, r.cacheTTL); found {
			if resp, err := unmarshalResponse(data); err == nil {
				if r.logging {
					slog.Debug("HTTP cache hit",
						"method", r.method,
						"url", r.url,
						"cache_key", cacheKey,
					)
				}
				return resp, nil
			}
		}
	}

	// Apply timeout to the context right before execution
	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Build the URL with query parameters
	reqURL, err := r.buildURL()
	if err != nil {
		return nil, err
	}

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, r.method, reqURL, r.body)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header = r.headers

	// Log request if enabled
	start := time.Now()
	if r.logging {
		r.logRequest(req)
	}

	// Get the client
	client := r.client
	if client == nil {
		client = DefaultClient()
	}

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Build the response
	result := &response.Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header,
		Body:       body,
	}

	// Log response if enabled
	if r.logging {
		r.logResponse(result, time.Since(start))
	}

	// Cache successful GET responses
	if r.method == http.MethodGet && r.cacheStrategy != cacheutils.None && result.IsSuccess() {
		cacheKey := r.buildCacheKey()
		if data, err := marshalResponse(result); err == nil {
			if err := cacheutils.Store(cacheKey, data, r.cacheStrategy, r.cacheTTL); err != nil {
				slog.Debug("Failed to cache response",
					"error", err,
					"cache_key", cacheKey,
				)
			}
		}
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

// buildCacheKey generates a cache key from URL and query parameters.
func (r *Request) buildCacheKey() string {
	reqURL, _ := r.buildURL()
	return reqURL
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
