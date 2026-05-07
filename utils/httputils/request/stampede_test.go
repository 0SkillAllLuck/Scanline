package request

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0skillallluck/scanline/utils/cacheutils"
)

// TestSingleflight_DedupesConcurrentCacheMisses verifies that N concurrent
// callers requesting the same uncached URL produce a single backend fetch.
func TestSingleflight_DedupesConcurrentCacheMisses(t *testing.T) {
	cacheutils.Clear() //nolint:errcheck

	var serverHits int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&serverHits, 1)
		// Hold long enough for concurrent callers to pile up on the
		// singleflight slot.
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"k":"v"}`))
	}))
	defer server.Close()

	const concurrent = 20
	var wg sync.WaitGroup
	wg.Add(concurrent)

	bodies := make([][]byte, concurrent)
	for i := 0; i < concurrent; i++ {
		go func(i int) {
			defer wg.Done()
			resp, err := NewRequest(http.MethodGet, server.URL).
				WithInMemoryCaching(60).
				Do()
			if err != nil {
				t.Errorf("goroutine %d: %v", i, err)
				return
			}
			bodies[i] = resp.Body
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt64(&serverHits); got != 1 {
		t.Errorf("expected 1 server hit, got %d", got)
	}

	for i, b := range bodies {
		if string(b) != `{"k":"v"}` {
			t.Errorf("goroutine %d body = %q, want %q", i, b, `{"k":"v"}`)
		}
	}
}

// TestSingleflight_FollowerHonorsContextCancellation verifies that a follower
// waiting on a slow leader's fetch can abandon its wait when its own context
// is canceled, instead of blocking until the leader returns.
func TestSingleflight_FollowerHonorsContextCancellation(t *testing.T) {
	cacheutils.Clear() //nolint:errcheck

	leaderStarted := make(chan struct{})
	releaseLeader := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case leaderStarted <- struct{}{}:
		default:
		}
		<-releaseLeader
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	defer close(releaseLeader)

	// Leader: long timeout, will block until releaseLeader fires.
	leaderDone := make(chan error, 1)
	go func() {
		_, err := NewRequest(http.MethodGet, server.URL).WithInMemoryCaching(60).Do()
		leaderDone <- err
	}()

	// Wait for the leader's request to actually hit the server before
	// dispatching the follower, so the follower lands on the in-flight slot.
	<-leaderStarted

	// Follower: short context, should abandon long before the leader returns.
	followerCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := NewRequest(http.MethodGet, server.URL).
		WithContext(followerCtx).
		WithInMemoryCaching(60).
		Do()
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("follower should have errored out via ctx cancellation")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("follower err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("follower waited %v — should have abandoned at the 100ms deadline", elapsed)
	}
}

// TestDoCached_CorruptCacheTriggersRefetch verifies that bytes that cannot be
// unmarshaled into a response (e.g. residue from a previous schema or disk
// corruption) trigger a fresh fetch instead of bubbling up the unmarshal error.
func TestDoCached_CorruptCacheTriggersRefetch(t *testing.T) {
	cacheutils.SetFileCacheDir(t.TempDir())
	cacheutils.Clear() //nolint:errcheck

	var serverHits int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&serverHits, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	// Compute the cache key the request layer will use, then plant garbage.
	r := NewRequest(http.MethodGet, server.URL).WithCaching(60)
	cacheKey := r.buildCacheKey()
	if err := cacheutils.Store(cacheKey, []byte("not a valid response"), cacheutils.Layered, 60); err != nil {
		t.Fatalf("Store garbage: %v", err)
	}

	resp, err := NewRequest(http.MethodGet, server.URL).WithCaching(60).Do()
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got := atomic.LoadInt64(&serverHits); got != 1 {
		t.Errorf("expected fallback to network: serverHits=%d", got)
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Errorf("body = %q, want fresh server response", resp.Body)
	}

	// And the corrupt entry must be gone — a follow-up call should hit the cache.
	if _, err := NewRequest(http.MethodGet, server.URL).WithCaching(60).Do(); err != nil {
		t.Fatalf("second Do: %v", err)
	}
	if got := atomic.LoadInt64(&serverHits); got != 1 {
		t.Errorf("expected second call to hit cache: serverHits=%d", got)
	}
}

// TestSingleflight_DoesNotShareResponseBuffers verifies that mutating one
// caller's response body doesn't affect siblings — i.e. each caller gets an
// independent unmarshaled response.
func TestSingleflight_DoesNotShareResponseBuffers(t *testing.T) {
	cacheutils.Clear() //nolint:errcheck

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()

	const concurrent = 5
	var wg sync.WaitGroup
	wg.Add(concurrent)
	results := make([]*[]byte, concurrent)

	for i := 0; i < concurrent; i++ {
		go func(i int) {
			defer wg.Done()
			resp, err := NewRequest(http.MethodGet, server.URL).WithInMemoryCaching(60).Do()
			if err != nil {
				t.Errorf("goroutine %d: %v", i, err)
				return
			}
			results[i] = &resp.Body
		}(i)
	}
	wg.Wait()

	// Mutate one caller's body and ensure others are unaffected.
	(*results[0])[0] = 'X'
	for i := 1; i < concurrent; i++ {
		if (*results[i])[0] != 'h' {
			t.Errorf("goroutine %d body was mutated by sibling: %q", i, *results[i])
		}
	}
}
