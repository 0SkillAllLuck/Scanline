package request

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWithClient_CustomClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	customClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	_, err := NewRequest(http.MethodGet, server.URL).
		WithClient(customClient).
		Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}
