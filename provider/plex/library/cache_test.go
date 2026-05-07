package library

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0skillallluck/scanline/provider/plex/base"
	"github.com/0skillallluck/scanline/utils/cacheutils"
)

// TestSections_CachingHonorsLibrariesPolicy verifies the libraries policy
// gates caching for Library.Sections end-to-end.
func TestSections_CachingHonorsLibrariesPolicy(t *testing.T) {
	cacheutils.SetFileCacheDir(t.TempDir())
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/library/sections/all") {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"MediaContainer":{"Directory":[]}}`))
	}))
	defer server.Close()

	cachingOn := true
	lib := New(&base.Base{
		BaseURL:  server.URL,
		Token:    "tok",
		ClientID: "cid",
		Cache: base.CachePolicy{
			Libraries: func() bool { return cachingOn },
			Metadata:  func() bool { return false },
		},
	})

	ctx := context.Background()

	if _, err := lib.Sections(ctx); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := lib.Sections(ctx); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected second call to hit cache: callCount=%d", callCount)
	}

	// Toggle caching off → next call must hit the server.
	cachingOn = false
	if _, err := lib.Sections(ctx); err != nil {
		t.Fatalf("third call: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected cache bypass after toggling off: callCount=%d", callCount)
	}
}

// TestMetadata_CachingScopedByToken verifies that two clients with different
// tokens against the same server URL do NOT share cached responses.
func TestMetadata_CachingScopedByToken(t *testing.T) {
	cacheutils.SetFileCacheDir(t.TempDir())
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"MediaContainer":{"Metadata":[{"ratingKey":"42","title":"Item"}]}}`))
	}))
	defer server.Close()

	always := func() bool { return true }
	libA := New(&base.Base{BaseURL: server.URL, Token: "tokA", ClientID: "cid", Cache: base.CachePolicy{Metadata: always}})
	libB := New(&base.Base{BaseURL: server.URL, Token: "tokB", ClientID: "cid", Cache: base.CachePolicy{Metadata: always}})

	ctx := context.Background()

	if _, err := libA.Metadata(ctx, "42"); err != nil {
		t.Fatalf("libA call: %v", err)
	}
	if _, err := libB.Metadata(ctx, "42"); err != nil {
		t.Fatalf("libB call: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected separate cache entries per token: callCount=%d", callCount)
	}

	// Repeat libA — should hit cache.
	if _, err := libA.Metadata(ctx, "42"); err != nil {
		t.Fatalf("libA repeat: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected libA repeat to be cached: callCount=%d", callCount)
	}
}

// TestMetadata_NoCachingWhenPolicyOff verifies that with the metadata policy
// disabled, every call hits the server.
func TestMetadata_NoCachingWhenPolicyOff(t *testing.T) {
	cacheutils.SetFileCacheDir(t.TempDir())
	cacheutils.Clear() //nolint:errcheck

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"MediaContainer":{"Metadata":[{"ratingKey":"42"}]}}`))
	}))
	defer server.Close()

	lib := New(&base.Base{
		BaseURL:  server.URL,
		Token:    "tok",
		ClientID: "cid",
	})

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := lib.Metadata(ctx, "42"); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if callCount != 3 {
		t.Errorf("expected no caching when policy is nil: callCount=%d", callCount)
	}
}
