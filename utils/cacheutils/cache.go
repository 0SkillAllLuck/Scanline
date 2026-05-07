package cacheutils

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"sync"
)

// Strategy specifies the caching strategy to use.
type Strategy int

const (
	None       Strategy = iota
	MemoryOnly          // In-memory only, no file persistence
	Layered             // Memory L1 + File L2 with promotion
)

// rawKeyIndex maps raw cache keys to their hashed counterparts (and back),
// supporting DeleteByPrefix since hashed keys are not prefix-comparable. The
// index is in-process only; on cold start prefix invalidation falls back to
// TTL expiry until entries are re-cached.
var (
	rawKeyMu       sync.RWMutex
	rawKeyIndex    = make(map[string]string) // raw → hashed
	hashedToRawKey = make(map[string]string) // hashed → raw
)

// Get retrieves cached data using the given strategy.
// For Layered strategy, checks memory first, then file (promoting hits to memory).
func Get(key string, strategy Strategy, ttl int) ([]byte, bool) {
	if strategy == None {
		return nil, false
	}

	hashedKey := hashKey(key)

	// Always check memory first
	if data, ok := getFromMemory(hashedKey); ok {
		return data, true
	}

	// For memory-only, we're done
	if strategy == MemoryOnly {
		return nil, false
	}

	// Check file cache (Layered)
	data, ok := getFromFile(hashedKey, ttl)
	if !ok {
		return nil, false
	}

	// Re-index on promotion: rawKeyIndex is in-process only, so file hits
	// across restarts need re-adding for DeleteByPrefix to find them.
	storeInMemory(hashedKey, data, ttl, true)
	rememberRawKey(key, hashedKey)

	return data, true
}

// Store stores data using the given strategy.
// For Layered strategy, stores in both memory and file.
func Store(key string, data []byte, strategy Strategy, ttl int) error {
	if strategy == None {
		return nil
	}

	hashedKey := hashKey(key)
	rememberRawKey(key, hashedKey)

	// Always store in memory
	storeInMemory(hashedKey, data, ttl, strategy == Layered)

	// For memory-only, we're done
	if strategy == MemoryOnly {
		return nil
	}

	// Store in file (Layered)
	return storeInFile(hashedKey, data, ttl)
}

// Delete removes a cache entry from all layers.
func Delete(key string) {
	hashedKey := hashKey(key)
	deleteFromMemory(hashedKey)
	deleteFromFile(hashedKey)
	forgetRawKey(key)
}

// DeleteByPrefix removes every cache entry whose raw key starts with prefix.
// Entries Stored between the scan and the delete phase aren't affected —
// they're newer than the invalidation request.
func DeleteByPrefix(prefix string) {
	rawKeyMu.RLock()
	matches := make([]struct{ raw, hashed string }, 0)
	for raw, hashed := range rawKeyIndex {
		if strings.HasPrefix(raw, prefix) {
			matches = append(matches, struct{ raw, hashed string }{raw, hashed})
		}
	}
	rawKeyMu.RUnlock()

	if len(matches) == 0 {
		return
	}

	rawKeyMu.Lock()
	for _, m := range matches {
		delete(rawKeyIndex, m.raw)
		delete(hashedToRawKey, m.hashed)
	}
	rawKeyMu.Unlock()

	for _, m := range matches {
		deleteFromMemory(m.hashed)
		deleteFromFile(m.hashed)
	}
}

// Clear removes all cache entries from all layers.
func Clear() error {
	clearMemory()
	clearRawKeyIndex()
	return clearFileDir()
}

func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return base64.URLEncoding.EncodeToString(hash[:])
}

func rememberRawKey(raw, hashed string) {
	rawKeyMu.Lock()
	rawKeyIndex[raw] = hashed
	hashedToRawKey[hashed] = raw
	rawKeyMu.Unlock()
}

func forgetRawKey(raw string) {
	rawKeyMu.Lock()
	if hashed, ok := rawKeyIndex[raw]; ok {
		delete(rawKeyIndex, raw)
		delete(hashedToRawKey, hashed)
	}
	rawKeyMu.Unlock()
}

// forgetHashedKey drops the index entry for a given hashed key. Called from
// LRU eviction so the index doesn't outlive the data.
func forgetHashedKey(hashed string) {
	rawKeyMu.Lock()
	if raw, ok := hashedToRawKey[hashed]; ok {
		delete(hashedToRawKey, hashed)
		delete(rawKeyIndex, raw)
	}
	rawKeyMu.Unlock()
}

func clearRawKeyIndex() {
	rawKeyMu.Lock()
	rawKeyIndex = make(map[string]string)
	hashedToRawKey = make(map[string]string)
	rawKeyMu.Unlock()
}
