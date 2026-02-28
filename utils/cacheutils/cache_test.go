package cacheutils

import (
	"os"
	"testing"
	"time"
)

func TestMemoryOnly_StoresAndRetrieves(t *testing.T) {
	Clear()

	data := []byte(`{"data":"test"}`)
	key := "test-memory-key"

	err := Store(key, data, MemoryOnly, 0)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	got, found := Get(key, MemoryOnly, 0)
	if !found {
		t.Fatal("Get() found = false, want true")
	}

	if string(got) != string(data) {
		t.Errorf("Data = %s, want %s", got, data)
	}
}

func TestMemoryOnly_PersistsWithoutExpiration(t *testing.T) {
	Clear()

	data := []byte("persistent data")
	key := "persistent-key"

	err := Store(key, data, MemoryOnly, 0)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	got, found := Get(key, MemoryOnly, 0)
	if !found {
		t.Fatal("In-memory cache should persist without expiration")
	}
	if string(got) != string(data) {
		t.Errorf("Data = %s, want %s", got, data)
	}
}

func TestLayered_StoresAndRetrieves(t *testing.T) {
	oldCacheDir := fileCacheDir
	SetFileCacheDir(t.TempDir())
	defer func() { fileCacheDir = oldCacheDir }()

	Clear()

	data := []byte(`{"data":"layered"}`)
	key := "test-layered-key"
	ttl := 60

	err := Store(key, data, Layered, ttl)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	got, found := Get(key, Layered, ttl)
	if !found {
		t.Fatal("Get() found = false, want true")
	}

	if string(got) != string(data) {
		t.Errorf("Data = %s, want %s", got, data)
	}
}

func TestLayered_FileHitPromotesToMemory(t *testing.T) {
	oldCacheDir := fileCacheDir
	SetFileCacheDir(t.TempDir())
	defer func() { fileCacheDir = oldCacheDir }()

	Clear()

	data := []byte(`{"data":"promote-me"}`)
	key := "promote-key"
	hashedKey := hashKey(key)
	ttl := 60

	// Store directly in file only (bypass public API)
	err := storeInFile(hashedKey, data, ttl)
	if err != nil {
		t.Fatalf("storeInFile() error = %v", err)
	}

	// Verify not in memory
	_, inMemory := getFromMemory(hashedKey)
	if inMemory {
		t.Fatal("Should not be in memory before Get()")
	}

	// Get via layered should find in file and promote
	got, found := Get(key, Layered, ttl)
	if !found {
		t.Fatal("Get() should find file-cached entry")
	}
	if string(got) != string(data) {
		t.Errorf("Data = %s, want %s", got, data)
	}

	// Now it should be in memory
	memData, inMemory := getFromMemory(hashedKey)
	if !inMemory {
		t.Fatal("File hit should have been promoted to memory")
	}
	if string(memData) != string(data) {
		t.Errorf("Promoted data = %s, want %s", memData, data)
	}
}

func TestLayered_MemoryHitSkipsFile(t *testing.T) {
	oldCacheDir := fileCacheDir
	SetFileCacheDir(t.TempDir())
	defer func() { fileCacheDir = oldCacheDir }()

	Clear()

	data := []byte(`{"data":"memory-hit"}`)
	key := "memory-hit-key"
	hashedKey := hashKey(key)

	// Store only in memory
	storeInMemory(hashedKey, data)

	// Get should return from memory without needing file
	got, found := Get(key, Layered, 60)
	if !found {
		t.Fatal("Get() should find memory-cached entry")
	}
	if string(got) != string(data) {
		t.Errorf("Data = %s, want %s", got, data)
	}
}

func TestLayered_ExpiresAfterTTL(t *testing.T) {
	oldCacheDir := fileCacheDir
	SetFileCacheDir(t.TempDir())
	defer func() { fileCacheDir = oldCacheDir }()

	Clear()

	data := []byte("expiring data")
	key := "expiring-key"
	ttl := 1

	err := Store(key, data, Layered, ttl)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	_, found := Get(key, Layered, ttl)
	if !found {
		t.Fatal("Get() should find cache immediately after storing")
	}

	// Clear memory so we rely on file cache for expiry test
	clearMemory()

	time.Sleep(2 * time.Second)

	_, found = Get(key, Layered, ttl)
	if found {
		t.Error("Get() should not find expired cache entry")
	}
}

func TestLayered_IndefiniteTTL(t *testing.T) {
	oldCacheDir := fileCacheDir
	SetFileCacheDir(t.TempDir())
	defer func() { fileCacheDir = oldCacheDir }()

	Clear()

	data := []byte("indefinite data")
	key := "indefinite-key"

	err := Store(key, data, Layered, 0)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	got, found := Get(key, Layered, 0)
	if !found {
		t.Fatal("Get() should find cache with TTL=0 (indefinite)")
	}
	if string(got) != string(data) {
		t.Errorf("Data = %s, want %s", got, data)
	}
}

func TestClear_RemovesAll(t *testing.T) {
	oldCacheDir := fileCacheDir
	SetFileCacheDir(t.TempDir())
	defer func() { fileCacheDir = oldCacheDir }()

	Clear()

	data := []byte("test")

	_ = Store("memory-key-1", data, MemoryOnly, 0)
	_ = Store("memory-key-2", data, MemoryOnly, 0)
	_ = Store("layered-key-1", data, Layered, 60)

	err := Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	_, found := Get("memory-key-1", MemoryOnly, 0)
	if found {
		t.Error("Memory cache should be cleared")
	}

	_, found = Get("memory-key-2", MemoryOnly, 0)
	if found {
		t.Error("Memory cache should be cleared")
	}
}

func TestDelete_RemovesFromAllLayers(t *testing.T) {
	oldCacheDir := fileCacheDir
	SetFileCacheDir(t.TempDir())
	defer func() { fileCacheDir = oldCacheDir }()

	Clear()

	data := []byte("deletable")
	key := "delete-me"

	_ = Store(key, data, Layered, 60)

	Delete(key)

	_, found := Get(key, Layered, 60)
	if found {
		t.Error("Delete() should remove from all layers")
	}
}

func TestHashKey_ConsistentHashing(t *testing.T) {
	key1 := hashKey("test-key")
	key2 := hashKey("test-key")

	if key1 != key2 {
		t.Errorf("Same input should produce same hash: %s != %s", key1, key2)
	}

	key3 := hashKey("different-key")
	if key1 == key3 {
		t.Error("Different inputs should produce different hashes")
	}
}

func TestNone_NeverCaches(t *testing.T) {
	data := []byte("no-cache")

	err := Store("none-key", data, None, 0)
	if err != nil {
		t.Fatalf("Store() with None should not error: %v", err)
	}

	_, found := Get("none-key", None, 0)
	if found {
		t.Error("Get() with None should never find cached data")
	}
}

func TestSetFileCacheDir(t *testing.T) {
	oldCacheDir := fileCacheDir
	defer func() { fileCacheDir = oldCacheDir }()

	tmpDir := os.TempDir()
	SetFileCacheDir(tmpDir)
	if fileCacheDir != tmpDir {
		t.Errorf("SetFileCacheDir() = %s, want %s", fileCacheDir, tmpDir)
	}
}
