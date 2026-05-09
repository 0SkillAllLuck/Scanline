package secrets

import (
	"errors"
	"sync"

	"github.com/0skillallluck/scanline/internal/g"
)

var (
	ErrKeyNotFound = errors.New("key not found")
)

// getService returns the platform secrets service wrapped in an in-memory cache.
// The cache makes each unique key cost at most one keychain prompt per process,
// which matters on macOS where unsigned binaries re-prompt on every access.
var getService = g.Lazy(func() Service {
	return &cachedService{
		inner: newService(),
		items: make(map[string]cacheEntry),
	}
})

type Service interface {
	Available() *ServiceError
	Delete(key string) error
	Get(key string) (Item, error)
	Has(key string) (bool, error)
	Set(key string, value Item) error
}

type Item struct {
	Label    string
	Password string
}

type ServiceError struct {
	Title string
	Body  string
	Fatal bool
}

func Healthcheck() *ServiceError {
	return getService().Available()
}

type cacheEntry struct {
	item  Item
	found bool // false → memoized ErrKeyNotFound
}

type cachedService struct {
	inner Service
	mu    sync.RWMutex
	items map[string]cacheEntry
}

func (c *cachedService) Available() *ServiceError {
	return c.inner.Available()
}

func (c *cachedService) Get(key string) (Item, error) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if ok {
		if !entry.found {
			return Item{}, ErrKeyNotFound
		}
		return entry.item, nil
	}

	// Call the keychain unlocked so concurrent misses for distinct keys
	// (e.g. RefreshServers reading one token per account) parallelize
	// instead of serializing. Concurrent misses for the same key may both
	// fetch — acceptable here since no caller reads the same key twice in
	// parallel.
	item, err := c.inner.Get(key)
	if err == ErrKeyNotFound {
		c.mu.Lock()
		c.items[key] = cacheEntry{found: false}
		c.mu.Unlock()
		return Item{}, ErrKeyNotFound
	}
	if err != nil {
		// Transient errors may resolve on retry; caching would mask recovery.
		return Item{}, err
	}
	c.mu.Lock()
	c.items[key] = cacheEntry{item: item, found: true}
	c.mu.Unlock()
	return item, nil
}

func (c *cachedService) Has(key string) (bool, error) {
	_, err := c.Get(key)
	if err == ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *cachedService) Set(key string, value Item) error {
	if err := c.inner.Set(key, value); err != nil {
		return err
	}
	c.mu.Lock()
	c.items[key] = cacheEntry{item: value, found: true}
	c.mu.Unlock()
	return nil
}

func (c *cachedService) Delete(key string) error {
	err := c.inner.Delete(key)
	c.mu.Lock()
	if err == nil || err == ErrKeyNotFound {
		c.items[key] = cacheEntry{found: false}
	} else {
		// Real error — invalidate so the next Get refetches.
		delete(c.items, key)
	}
	c.mu.Unlock()
	return err
}
