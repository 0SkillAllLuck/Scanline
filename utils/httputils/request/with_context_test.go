package request

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWithContext_CancellationPropagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := NewRequest(http.MethodGet, server.URL).
		WithContext(ctx).
		Do()

	if err == nil {
		t.Error("Expected context cancellation error")
	}
}
