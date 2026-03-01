package request

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0skillallluck/scanline/utils/cacheutils"
)

func TestWithCaching_GetRequest(t *testing.T) {
	cacheutils.SetFileCacheDir(t.TempDir())
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"call":` + string(rune('0'+callCount)) + `}`)) //nolint:errcheck
	}))
	defer server.Close()

	resp1, err := NewRequest(http.MethodGet, server.URL).
		WithCaching(60).
		Do()
	if err != nil {
		t.Fatalf("First Do() error = %v", err)
	}

	resp2, err := NewRequest(http.MethodGet, server.URL).
		WithCaching(60).
		Do()
	if err != nil {
		t.Fatalf("Second Do() error = %v", err)
	}

	if callCount != 1 {
		t.Errorf("Server was called %d times, want 1 (second should be cached)", callCount)
	}

	if string(resp1.Body) != string(resp2.Body) {
		t.Errorf("Cached response differs from original")
	}

	// Clean up
	cacheutils.Delete(server.URL)
}

func TestWithCaching_IndefiniteTTL(t *testing.T) {
	cacheutils.SetFileCacheDir(t.TempDir())
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":"indefinite"}`)) //nolint:errcheck
	}))
	defer server.Close()

	resp1, err := NewRequest(http.MethodGet, server.URL).
		WithCaching(0). // TTL=0 means indefinite
		Do()
	if err != nil {
		t.Fatalf("First Do() error = %v", err)
	}

	resp2, err := NewRequest(http.MethodGet, server.URL).
		WithCaching(0).
		Do()
	if err != nil {
		t.Fatalf("Second Do() error = %v", err)
	}

	if callCount != 1 {
		t.Errorf("Server was called %d times, want 1 (second should be cached)", callCount)
	}

	if string(resp1.Body) != string(resp2.Body) {
		t.Errorf("Cached response differs from original")
	}

	// Clean up
	cacheutils.Delete(server.URL)
}

func TestWithCaching_ExpiresAfterTTL(t *testing.T) {
	cacheutils.SetFileCacheDir(t.TempDir())
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"call":` + string(rune('0'+callCount)) + `}`)) //nolint:errcheck
	}))
	defer server.Close()

	_, err := NewRequest(http.MethodGet, server.URL).
		WithCaching(1). // 1 second TTL
		Do()
	if err != nil {
		t.Fatalf("First Do() error = %v", err)
	}

	// Clear memory so expiry test relies on file TTL
	cacheutils.Clear() //nolint:errcheck

	time.Sleep(2 * time.Second)

	_, err = NewRequest(http.MethodGet, server.URL).
		WithCaching(1).
		Do()
	if err != nil {
		t.Fatalf("Second Do() error = %v", err)
	}

	if callCount != 2 {
		t.Errorf("Server was called %d times, want 2 (cache should have expired)", callCount)
	}

	// Clean up
	cacheutils.Delete(server.URL)
}

func TestWithCaching_NonGetRequestIgnored(t *testing.T) {
	cacheutils.SetFileCacheDir(t.TempDir())
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _ = NewRequest(http.MethodPost, server.URL).WithCaching(60).Do()
	_, _ = NewRequest(http.MethodPost, server.URL).WithCaching(60).Do()

	if callCount != 2 {
		t.Errorf("POST requests should not be cached, got %d calls instead of 2", callCount)
	}
}
