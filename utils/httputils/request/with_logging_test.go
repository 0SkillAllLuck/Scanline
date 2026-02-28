package request

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithLogging_RedactsHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// This test mainly verifies that logging doesn't panic and redaction is configured
	_, err := NewRequest(http.MethodGet, server.URL).
		WithHeader("Authorization", "secret-token").
		WithHeader("X-Api-Key", "api-key-value").
		WithLogging("Authorization", "X-Api-Key").
		Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}
