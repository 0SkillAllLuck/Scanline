package request

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithJSONBody(t *testing.T) {
	type requestBody struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %s, want POST", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", contentType)
		}

		body, _ := io.ReadAll(r.Body)
		var got requestBody
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("Failed to unmarshal request body: %v", err)
		}

		if got.Name != "test" || got.Value != 42 {
			t.Errorf("Body = %+v, want {Name:test Value:42}", got)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	body := requestBody{Name: "test", Value: 42}
	resp, err := NewRequest(http.MethodPost, server.URL).WithJSONBody(body).Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if resp.StatusCode != 201 {
		t.Errorf("StatusCode = %d, want 201", resp.StatusCode)
	}
}

func TestWithBody_RawBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "raw body content" {
			t.Errorf("Body = %s, want 'raw body content'", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := NewRequest(http.MethodPost, server.URL).
		WithBody(bytes.NewReader([]byte("raw body content"))).
		Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}

func TestWithJSONBody_ErrorAccumulation(t *testing.T) {
	req := NewRequest(http.MethodPost, "http://example.com").
		WithJSONBody(make(chan int)) // channels cannot be marshaled

	_, err := req.Do()
	if err == nil {
		t.Error("Expected error from invalid JSON body")
	}
}
