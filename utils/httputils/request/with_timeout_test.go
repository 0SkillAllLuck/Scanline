package request

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWithTimeout_RequestTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest(http.MethodGet, server.URL).
		WithTimeout(100 * time.Millisecond).
		Do()

	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestWithTimeout_RequestSucceeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	resp, err := NewRequest(http.MethodGet, server.URL).
		WithTimeout(5 * time.Second).
		Do()

	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}
