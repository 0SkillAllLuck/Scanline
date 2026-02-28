package request

import (
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	defaultClient   *http.Client
	defaultClientMu sync.RWMutex
)

// DefaultClient returns the default HTTP client with sensible defaults.
// The client has connection pooling enabled and transport-level timeouts.
// Request-level timeouts should be set via WithTimeout() which uses context cancellation.
func DefaultClient() *http.Client {
	defaultClientMu.RLock()
	if defaultClient != nil {
		c := defaultClient
		defaultClientMu.RUnlock()
		return c
	}
	defaultClientMu.RUnlock()

	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()

	// Double-check after acquiring write lock
	if defaultClient != nil {
		return defaultClient
	}

	// Note: No http.Client.Timeout is set here intentionally.
	// Request timeouts should be controlled via WithTimeout() using context.
	// This avoids conflicts between client-level and context-level timeouts.
	defaultClient = &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second, // Connection establishment timeout
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second, // TLS handshake timeout
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	return defaultClient
}

// SetDefaultClient sets the default HTTP client.
// This is useful for testing or custom configurations.
func SetDefaultClient(client *http.Client) {
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	defaultClient = client
}
