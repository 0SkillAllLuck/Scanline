package request

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGet_BasicRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Method = %s, want GET", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"success"}`)) //nolint:errcheck
	}))
	defer server.Close()

	resp, err := NewRequest(http.MethodGet, server.URL).Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestPut_Request(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Method = %s, want PUT", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest(http.MethodPut, server.URL).Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}

func TestPatch_Request(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Method = %s, want PATCH", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest(http.MethodPatch, server.URL).Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}

func TestDelete_Request(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest(http.MethodDelete, server.URL).Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}

func TestDo_ReturnsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	}))
	defer server.Close()

	resp, err := NewRequest(http.MethodDelete, server.URL).Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	if resp.Headers.Get("X-Custom-Header") != "custom-value" {
		t.Errorf("X-Custom-Header = %s, want custom-value", resp.Headers.Get("X-Custom-Header"))
	}

	if string(resp.Body) != `{"status":"ok"}` {
		t.Errorf("Body = %s, want {\"status\":\"ok\"}", resp.Body)
	}
}

func TestDoAndDecode_ParsesJSON(t *testing.T) {
	type responseStruct struct {
		Message string `json:"message"`
		Count   int    `json:"count"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"hello","count":5}`)) //nolint:errcheck
	}))
	defer server.Close()

	var result responseStruct
	err := NewRequest(http.MethodDelete, server.URL).DoAndDecode(&result)
	if err != nil {
		t.Fatalf("DoAndDecode() error = %v", err)
	}

	if result.Message != "hello" {
		t.Errorf("Message = %s, want hello", result.Message)
	}
	if result.Count != 5 {
		t.Errorf("Count = %d, want 5", result.Count)
	}
}

func TestDoAndDecode_NonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`)) //nolint:errcheck
	}))
	defer server.Close()

	var result map[string]string
	err := NewRequest(http.MethodDelete, server.URL).DoAndDecode(&result)
	if err == nil {
		t.Error("DoAndDecode should return error for non-2xx status")
	}

	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Error should contain status code, got: %v", err)
	}
}
