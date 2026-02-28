package cacheutils

import "sync"

var (
	memoryCache   = make(map[string][]byte)
	memoryCacheMu sync.RWMutex
)

func getFromMemory(hashedKey string) ([]byte, bool) {
	memoryCacheMu.RLock()
	defer memoryCacheMu.RUnlock()

	data, ok := memoryCache[hashedKey]
	if !ok {
		return nil, false
	}

	return copyBytes(data), true
}

func storeInMemory(hashedKey string, data []byte) {
	memoryCacheMu.Lock()
	defer memoryCacheMu.Unlock()

	memoryCache[hashedKey] = copyBytes(data)
}

func deleteFromMemory(hashedKey string) {
	memoryCacheMu.Lock()
	defer memoryCacheMu.Unlock()

	delete(memoryCache, hashedKey)
}

func clearMemory() {
	memoryCacheMu.Lock()
	defer memoryCacheMu.Unlock()

	memoryCache = make(map[string][]byte)
}

func copyBytes(b []byte) []byte {
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp
}
