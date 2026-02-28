package request

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithHeader_SingleHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer token123" {
			t.Errorf("Authorization = %s, want Bearer token123", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest(http.MethodGet, server.URL).
		WithHeader("Authorization", "Bearer token123").
		Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}

func TestWithHeaders_MultipleHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-1") != "value1" {
			t.Errorf("X-Custom-1 = %s, want value1", r.Header.Get("X-Custom-1"))
		}
		if r.Header.Get("X-Custom-2") != "value2" {
			t.Errorf("X-Custom-2 = %s, want value2", r.Header.Get("X-Custom-2"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest(http.MethodGet, server.URL).
		WithHeaders(map[string]string{
			"X-Custom-1": "value1",
			"X-Custom-2": "value2",
		}).
		Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}
