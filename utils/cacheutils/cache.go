package cacheutils

import (
	"crypto/sha256"
	"encoding/base64"
)

// Strategy specifies the caching strategy to use.
type Strategy int

const (
	None       Strategy = iota
	MemoryOnly          // In-memory only, no file persistence
	Layered             // Memory L1 + File L2 with promotion
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

	// Promote file hit to memory
	storeInMemory(hashedKey, data)

	return data, true
}

// Store stores data using the given strategy.
// For Layered strategy, stores in both memory and file.
func Store(key string, data []byte, strategy Strategy, ttl int) error {
	if strategy == None {
		return nil
	}

	hashedKey := hashKey(key)

	// Always store in memory
	storeInMemory(hashedKey, data)

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
}

// Clear removes all cache entries from all layers.
func Clear() error {
	clearMemory()
	return clearFileDir()
}

func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return base64.URLEncoding.EncodeToString(hash[:])
}
