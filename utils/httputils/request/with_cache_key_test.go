package request

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0skillallluck/scanline/utils/cacheutils"
)

func TestWithCacheKey_DifferentExtrasProduceDifferentEntries(t *testing.T) {
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	_, err := NewRequest(http.MethodGet, server.URL).
		WithInMemoryCaching(0).
		WithCacheKey("user-A").
		Do()
	if err != nil {
		t.Fatalf("first request: %v", err)
	}

	_, err = NewRequest(http.MethodGet, server.URL).
		WithInMemoryCaching(0).
		WithCacheKey("user-B").
		Do()
	if err != nil {
		t.Fatalf("second request: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 server calls (one per cache key extra), got %d", callCount)
	}
}

func TestWithCacheKey_SameExtraReusesCacheEntry(t *testing.T) {
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	for i := 0; i < 3; i++ {
		_, err := NewRequest(http.MethodGet, server.URL).
			WithInMemoryCaching(0).
			WithCacheKey("same-token").
			Do()
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
	}

	if callCount != 1 {
		t.Errorf("expected 1 server call (subsequent requests cached), got %d", callCount)
	}
}

func TestWithInMemoryCaching_TTLExpires(t *testing.T) {
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	_, err := NewRequest(http.MethodGet, server.URL).WithInMemoryCaching(1).Do()
	if err != nil {
		t.Fatalf("first request: %v", err)
	}

	if _, err := NewRequest(http.MethodGet, server.URL).WithInMemoryCaching(1).Do(); err != nil {
		t.Fatalf("second request: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected cache hit on second request: callCount=%d", callCount)
	}

	// Wait past the TTL.
	time.Sleep(1100 * time.Millisecond)

	if _, err := NewRequest(http.MethodGet, server.URL).WithInMemoryCaching(1).Do(); err != nil {
		t.Fatalf("third request: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected cache miss after TTL expiry: callCount=%d", callCount)
	}
}
