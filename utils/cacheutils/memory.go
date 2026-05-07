package cacheutils

import (
	"container/list"
	"sync"
	"time"
)

// MemoryCacheMaxBytes is the upper bound on bytes held in the in-memory cache.
// When exceeded, least-recently-used entries are evicted until the total fits.
const MemoryCacheMaxBytes = 64 * 1024 * 1024 // 64 MiB

type memoryEntry struct {
	key       string
	data      []byte
	expiresAt time.Time // zero value = no expiration
	// layered: a file copy exists, so eviction must keep the rawKeyIndex
	// entry alive for DeleteByPrefix to reach it.
	layered bool
}

var (
	memoryMu    sync.Mutex
	memoryList  = list.New()
	memoryIndex = make(map[string]*list.Element)
	memoryBytes int64
)

// getFromMemory returns cached bytes if present and not expired, promoting the
// entry to the front of the LRU list.
func getFromMemory(hashedKey string) ([]byte, bool) {
	memoryMu.Lock()
	defer memoryMu.Unlock()

	el, ok := memoryIndex[hashedKey]
	if !ok {
		return nil, false
	}

	entry := el.Value.(*memoryEntry)
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		removeMemoryElement(el)
		return nil, false
	}

	memoryList.MoveToFront(el)
	return copyBytes(entry.data), true
}

// storeInMemory stores data under the given hashed key. ttl == 0 means no
// expiration (still subject to LRU). The byte cap is soft for the just-
// inserted entry — an oversized entry survives at least until the next Store,
// keeping the cache useful when a single response exceeds the cap.
func storeInMemory(hashedKey string, data []byte, ttl int, layered bool) {
	memoryMu.Lock()
	defer memoryMu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(time.Duration(ttl) * time.Second)
	}

	var inserted *list.Element
	if el, ok := memoryIndex[hashedKey]; ok {
		entry := el.Value.(*memoryEntry)
		memoryBytes -= int64(len(entry.data))
		entry.data = copyBytes(data)
		entry.expiresAt = expiresAt
		entry.layered = layered
		memoryBytes += int64(len(entry.data))
		memoryList.MoveToFront(el)
		inserted = el
	} else {
		entry := &memoryEntry{
			key:       hashedKey,
			data:      copyBytes(data),
			expiresAt: expiresAt,
			layered:   layered,
		}
		inserted = memoryList.PushFront(entry)
		memoryIndex[hashedKey] = inserted
		memoryBytes += int64(len(entry.data))
	}

	for memoryBytes > MemoryCacheMaxBytes {
		oldest := memoryList.Back()
		if oldest == nil || oldest == inserted {
			break
		}
		removeMemoryElement(oldest)
	}
}

func deleteFromMemory(hashedKey string) {
	memoryMu.Lock()
	defer memoryMu.Unlock()

	if el, ok := memoryIndex[hashedKey]; ok {
		removeMemoryElement(el)
	}
}

func clearMemory() {
	memoryMu.Lock()
	defer memoryMu.Unlock()

	memoryList.Init()
	memoryIndex = make(map[string]*list.Element)
	memoryBytes = 0
}

// removeMemoryElement evicts an LRU entry. Memory-only entries also drop the
// rawKeyIndex; layered entries keep theirs so DeleteByPrefix can still reach
// the file copy. Caller must hold memoryMu.
func removeMemoryElement(el *list.Element) {
	entry := el.Value.(*memoryEntry)
	memoryList.Remove(el)
	delete(memoryIndex, entry.key)
	memoryBytes -= int64(len(entry.data))
	if !entry.layered {
		forgetHashedKey(entry.key)
	}
}

func copyBytes(b []byte) []byte {
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp
}
