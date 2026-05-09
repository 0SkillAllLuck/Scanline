package secrets

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

type fakeService struct {
	mu    sync.Mutex
	store map[string]Item
	gets  int64
	sets  int64
	dels  int64
	getErr error // injected error for the next Get if non-nil
}

func newFakeService() *fakeService {
	return &fakeService{store: make(map[string]Item)}
}

func (f *fakeService) Available() *ServiceError { return nil }

func (f *fakeService) Get(key string) (Item, error) {
	atomic.AddInt64(&f.gets, 1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		err := f.getErr
		f.getErr = nil
		return Item{}, err
	}
	item, ok := f.store[key]
	if !ok {
		return Item{}, ErrKeyNotFound
	}
	return item, nil
}

func (f *fakeService) Has(key string) (bool, error) {
	_, err := f.Get(key)
	if err == ErrKeyNotFound {
		return false, nil
	}
	return err == nil, err
}

func (f *fakeService) Set(key string, value Item) error {
	atomic.AddInt64(&f.sets, 1)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store[key] = value
	return nil
}

func (f *fakeService) Delete(key string) error {
	atomic.AddInt64(&f.dels, 1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.store[key]; !ok {
		return ErrKeyNotFound
	}
	delete(f.store, key)
	return nil
}

func newCached(inner Service) *cachedService {
	return &cachedService{inner: inner, items: make(map[string]cacheEntry)}
}

func TestCachedService_GetCachesHits(t *testing.T) {
	fake := newFakeService()
	fake.store["k"] = Item{Password: "v"}
	cache := newCached(fake)

	for i := 0; i < 5; i++ {
		got, err := cache.Get("k")
		if err != nil || got.Password != "v" {
			t.Fatalf("Get #%d: got=%v err=%v", i, got, err)
		}
	}
	if got := atomic.LoadInt64(&fake.gets); got != 1 {
		t.Errorf("inner.Get called %d times, want 1", got)
	}
}

func TestCachedService_GetCachesMisses(t *testing.T) {
	fake := newFakeService()
	cache := newCached(fake)

	for i := 0; i < 5; i++ {
		_, err := cache.Get("missing")
		if err != ErrKeyNotFound {
			t.Fatalf("Get #%d: err=%v, want ErrKeyNotFound", i, err)
		}
	}
	if got := atomic.LoadInt64(&fake.gets); got != 1 {
		t.Errorf("inner.Get called %d times, want 1 (negative-cache memoization)", got)
	}
}

func TestCachedService_TransientErrorNotCached(t *testing.T) {
	fake := newFakeService()
	cache := newCached(fake)

	fake.getErr = errors.New("boom")
	if _, err := cache.Get("k"); err == nil {
		t.Fatal("expected error")
	}
	// Next call should hit the inner service again (no cached error).
	fake.store["k"] = Item{Password: "v"}
	got, err := cache.Get("k")
	if err != nil || got.Password != "v" {
		t.Fatalf("got=%v err=%v", got, err)
	}
	if got := atomic.LoadInt64(&fake.gets); got != 2 {
		t.Errorf("inner.Get called %d times, want 2", got)
	}
}

func TestCachedService_SetIsWriteThrough(t *testing.T) {
	fake := newFakeService()
	cache := newCached(fake)

	if err := cache.Set("k", Item{Password: "v"}); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt64(&fake.sets); got != 1 {
		t.Errorf("inner.Set called %d times, want 1", got)
	}
	got, err := cache.Get("k")
	if err != nil || got.Password != "v" {
		t.Fatalf("got=%v err=%v", got, err)
	}
	if got := atomic.LoadInt64(&fake.gets); got != 0 {
		t.Errorf("inner.Get called %d times after Set, want 0 (Set should populate cache)", got)
	}
}

func TestCachedService_DeleteInvalidates(t *testing.T) {
	fake := newFakeService()
	fake.store["k"] = Item{Password: "v"}
	cache := newCached(fake)

	if _, err := cache.Get("k"); err != nil {
		t.Fatal(err)
	}
	if err := cache.Delete("k"); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Get("k"); err != ErrKeyNotFound {
		t.Fatalf("post-delete Get err=%v, want ErrKeyNotFound", err)
	}
	// inner.Get should only have been called once (initial); Delete memoizes
	// not-found so the post-delete Get is a cache hit.
	if got := atomic.LoadInt64(&fake.gets); got != 1 {
		t.Errorf("inner.Get called %d times, want 1", got)
	}
}

func TestCachedService_HasUsesCache(t *testing.T) {
	fake := newFakeService()
	fake.store["k"] = Item{Password: "v"}
	cache := newCached(fake)

	for i := 0; i < 3; i++ {
		ok, err := cache.Has("k")
		if err != nil || !ok {
			t.Fatalf("Has #%d: ok=%v err=%v", i, ok, err)
		}
	}
	if got := atomic.LoadInt64(&fake.gets); got != 1 {
		t.Errorf("inner.Get called %d times, want 1", got)
	}
}

func TestCachedService_ConcurrentDistinctKeys(t *testing.T) {
	fake := newFakeService()
	for i := 0; i < 32; i++ {
		fake.store[key(i)] = Item{Password: "v"}
	}
	cache := newCached(fake)

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if _, err := cache.Get(key(i)); err != nil {
					t.Errorf("Get(%s): %v", key(i), err)
				}
			}
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt64(&fake.gets); got != 32 {
		t.Errorf("inner.Get called %d times, want 32 (one per distinct key)", got)
	}
}

func key(i int) string {
	return string(rune('a'+i%26)) + string(rune('0'+i/26))
}
