package request

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithQuery_AppliesQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("page = %s, want 1", r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("limit = %s, want 10", r.URL.Query().Get("limit"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest(http.MethodGet, server.URL).
		WithQuery(map[string]string{
			"page":  "1",
			"limit": "10",
		}).
		Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}

func TestWithQuery_PreservesExistingParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("existing") != "value" {
			t.Errorf("existing = %s, want value", r.URL.Query().Get("existing"))
		}
		if r.URL.Query().Get("new") != "param" {
			t.Errorf("new = %s, want param", r.URL.Query().Get("new"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest(http.MethodGet, server.URL+"?existing=value").
		WithQuery(map[string]string{"new": "param"}).
		Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}
