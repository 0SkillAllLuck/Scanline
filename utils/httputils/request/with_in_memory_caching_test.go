package request

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0skillallluck/scanline/utils/cacheutils"
)

func TestWithInMemoryCaching_GetRequest(t *testing.T) {
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"call":` + string(rune('0'+callCount)) + `}`)) //nolint:errcheck
	}))
	defer server.Close()

	resp1, err := NewRequest(http.MethodGet, server.URL).
		WithInMemoryCaching().
		Do()
	if err != nil {
		t.Fatalf("First Do() error = %v", err)
	}

	resp2, err := NewRequest(http.MethodGet, server.URL).
		WithInMemoryCaching().
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
}

func TestWithInMemoryCaching_PersistsForSessionLifetime(t *testing.T) {
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":"persistent"}`)) //nolint:errcheck
	}))
	defer server.Close()

	// First request
	_, err := NewRequest(http.MethodGet, server.URL).
		WithInMemoryCaching().
		Do()
	if err != nil {
		t.Fatalf("First Do() error = %v", err)
	}

	// Multiple subsequent requests should all be cached
	for i := range 5 {
		_, err := NewRequest(http.MethodGet, server.URL).
			WithInMemoryCaching().
			Do()
		if err != nil {
			t.Fatalf("Request %d Do() error = %v", i+2, err)
		}
	}

	if callCount != 1 {
		t.Errorf("Server was called %d times, want 1 (all should be cached)", callCount)
	}
}

func TestWithInMemoryCaching_NonGetRequestIgnored(t *testing.T) {
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _ = NewRequest(http.MethodPost, server.URL).WithInMemoryCaching().Do()
	_, _ = NewRequest(http.MethodPost, server.URL).WithInMemoryCaching().Do()

	if callCount != 2 {
		t.Errorf("POST requests should not be cached, got %d calls instead of 2", callCount)
	}
}

func TestWithInMemoryCaching_DifferentURLsDifferentCache(t *testing.T) {
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"path":"` + r.URL.Path + `"}`)) //nolint:errcheck
	}))
	defer server.Close()

	_, _ = NewRequest(http.MethodGet, server.URL+"/path1").WithInMemoryCaching().Do()
	_, _ = NewRequest(http.MethodGet, server.URL+"/path2").WithInMemoryCaching().Do()
	_, _ = NewRequest(http.MethodGet, server.URL+"/path1").WithInMemoryCaching().Do()
	_, _ = NewRequest(http.MethodGet, server.URL+"/path2").WithInMemoryCaching().Do()

	if callCount != 2 {
		t.Errorf("Server was called %d times, want 2 (one per unique path)", callCount)
	}
}
