package cacheutils

import (
	"strings"
	"testing"
	"time"
)

func TestMemoryEntry_RespectsTTL(t *testing.T) {
	clearMemory()

	storeInMemory("ttl-key", []byte("data"), 1, false)

	if _, ok := getFromMemory("ttl-key"); !ok {
		t.Fatal("entry should be present immediately after storing")
	}

	time.Sleep(1100 * time.Millisecond)

	if _, ok := getFromMemory("ttl-key"); ok {
		t.Error("entry should have expired")
	}
}

func TestMemoryEntry_TTLZeroNeverExpires(t *testing.T) {
	clearMemory()

	storeInMemory("indef-key", []byte("forever"), 0, false)

	time.Sleep(50 * time.Millisecond)

	if _, ok := getFromMemory("indef-key"); !ok {
		t.Error("TTL=0 entry should not expire")
	}
}

func TestMemoryEntry_LRUEvictionRespectsByteCap(t *testing.T) {
	clearMemory()
	t.Cleanup(clearMemory)

	// Size each entry so 64 entries exactly fill the cap and the 65th
	// triggers eviction of the oldest. Derived from MemoryCacheMaxBytes
	// so the test stays correct if the cap changes.
	const fillCount = 64
	entrySize := MemoryCacheMaxBytes / fillCount
	payload := make([]byte, entrySize)
	for i := byte(0); i < fillCount+1; i++ {
		payload[0] = i // make each payload distinguishable
		storeInMemory(string([]byte{'k', i}), payload, 0, false)
	}

	if memoryBytes > MemoryCacheMaxBytes {
		t.Errorf("memory bytes %d exceeds cap %d", memoryBytes, MemoryCacheMaxBytes)
	}

	// First-inserted key should be gone.
	if _, ok := getFromMemory(string([]byte{'k', 0})); ok {
		t.Error("oldest entry should have been evicted")
	}

	// Most recent key should still be present.
	if _, ok := getFromMemory(string([]byte{'k', fillCount})); !ok {
		t.Error("most recent entry should still be present")
	}
}

func TestMemoryEntry_GetMovesToFront(t *testing.T) {
	clearMemory()
	t.Cleanup(clearMemory)

	// Fill the cache to capacity with fillCount entries sized to MemoryCacheMaxBytes/fillCount.
	const fillCount = 64
	entrySize := MemoryCacheMaxBytes / fillCount
	payload := make([]byte, entrySize)
	for i := byte(0); i < fillCount; i++ {
		storeInMemory(string([]byte{'k', i}), payload, 0, false)
	}

	// Touch entry 0 — it becomes most recently used; entry 1 takes its
	// place as the least recently used.
	if _, ok := getFromMemory(string([]byte{'k', 0})); !ok {
		t.Fatal("entry 0 should be present")
	}

	// Insert one more entry to push us over the cap. The LRU (entry 1,
	// since 0 was promoted) should be evicted.
	storeInMemory("new", payload, 0, false)

	if _, ok := getFromMemory(string([]byte{'k', 0})); !ok {
		t.Error("entry 0 should have survived because Get promoted it")
	}
	if _, ok := getFromMemory(string([]byte{'k', 1})); ok {
		t.Error("entry 1 should have been evicted as the LRU")
	}
}

func TestDeleteByPrefix_RemovesMatchingEntries(t *testing.T) {
	SetFileCacheDir(t.TempDir())
	Clear() //nolint:errcheck

	keys := []string{
		"https://server/library/metadata/123",
		"https://server/library/metadata/123?includeMarkers=1",
		"https://server/library/metadata/456",
		"https://server/hubs/promoted",
	}
	for _, k := range keys {
		_ = Store(k, []byte("v:"+k), Layered, 60)
	}

	DeleteByPrefix("https://server/library/metadata/123")

	for _, k := range keys {
		_, ok := Get(k, Layered, 60)
		shouldExist := !strings.HasPrefix(k, "https://server/library/metadata/123")
		if shouldExist && !ok {
			t.Errorf("non-matching key %q should still be present", k)
		}
		if !shouldExist && ok {
			t.Errorf("matching key %q should have been deleted", k)
		}
	}
}

func TestDeleteByPrefix_RemovesFromBothLayers(t *testing.T) {
	SetFileCacheDir(t.TempDir())
	Clear() //nolint:errcheck

	key := "https://server/foo/bar"
	_ = Store(key, []byte("v"), Layered, 60)

	// Verify present in both layers
	hashed := hashKey(key)
	if _, ok := getFromMemory(hashed); !ok {
		t.Fatal("expected entry in memory before DeleteByPrefix")
	}
	if _, ok := getFromFile(hashed, 60); !ok {
		t.Fatal("expected entry in file cache before DeleteByPrefix")
	}

	DeleteByPrefix("https://server/foo/")

	if _, ok := getFromMemory(hashed); ok {
		t.Error("entry should be gone from memory after DeleteByPrefix")
	}
	if _, ok := getFromFile(hashed, 60); ok {
		t.Error("entry should be gone from file cache after DeleteByPrefix")
	}
}

func TestLRUEviction_CleansRawKeyIndex(t *testing.T) {
	SetFileCacheDir(t.TempDir())
	Clear() //nolint:errcheck

	const fillCount = 64
	entrySize := MemoryCacheMaxBytes / fillCount
	payload := make([]byte, entrySize)

	// Fill exactly to capacity then push one over so a single eviction occurs.
	for i := byte(0); i < fillCount; i++ {
		_ = Store("key-"+string([]byte{i}), payload, MemoryOnly, 0)
	}
	_ = Store("key-evictor", payload, MemoryOnly, 0)

	rawKeyMu.RLock()
	rawCount := len(rawKeyIndex)
	hashedCount := len(hashedToRawKey)
	rawKeyMu.RUnlock()

	if rawCount > fillCount {
		t.Errorf("rawKeyIndex should not retain entries past LRU eviction: got %d", rawCount)
	}
	if rawCount != hashedCount {
		t.Errorf("rawKeyIndex (%d) and hashedToRawKey (%d) should stay in sync", rawCount, hashedCount)
	}
}

// TestFileHit_ReindexesRawKey simulates a process restart (clearing the
// in-process raw-key index and the memory cache while leaving the file cache
// intact). After a Get re-indexes the entry, DeleteByPrefix must be able to
// find and delete the file copy.
func TestFileHit_ReindexesRawKey(t *testing.T) {
	SetFileCacheDir(t.TempDir())
	Clear() //nolint:errcheck

	key := "https://server/library/metadata/42"
	if err := Store(key, []byte("v"), Layered, 600); err != nil {
		t.Fatalf("Store: %v", err)
	}

	// Simulate a process restart: in-memory state goes away, file cache survives.
	clearMemory()
	clearRawKeyIndex()

	// First read after "restart" — should hit the file and re-index.
	if _, ok := Get(key, Layered, 600); !ok {
		t.Fatal("expected file-cache hit after simulated restart")
	}

	rawKeyMu.RLock()
	hashed, indexed := rawKeyIndex[key]
	rawKeyMu.RUnlock()
	if !indexed {
		t.Fatal("file-hit Get should re-add the raw key to the index")
	}
	if hashed != hashKey(key) {
		t.Errorf("re-indexed mapping mismatch: got %q want %q", hashed, hashKey(key))
	}

	// Invalidation should now find and delete the file copy.
	DeleteByPrefix("https://server/library/metadata/")
	if _, ok := getFromFile(hashKey(key), 600); ok {
		t.Error("DeleteByPrefix should have removed the file-cache entry")
	}
}

// TestLayeredEviction_PreservesInvalidation verifies that LRU eviction of a
// Layered entry's memory copy keeps the rawKeyIndex entry intact so
// DeleteByPrefix can still reach the file copy.
func TestLayeredEviction_PreservesInvalidation(t *testing.T) {
	SetFileCacheDir(t.TempDir())
	Clear() //nolint:errcheck

	const fillCount = 64
	entrySize := MemoryCacheMaxBytes / fillCount
	smallPayload := []byte("v")
	bigPayload := make([]byte, entrySize)

	target := "https://server/library/metadata/42"
	if err := Store(target, smallPayload, Layered, 600); err != nil {
		t.Fatalf("Store target: %v", err)
	}

	// Push enough big entries through to evict the target from memory.
	for i := byte(0); i < fillCount+1; i++ {
		_ = Store("filler-"+string([]byte{i}), bigPayload, MemoryOnly, 0)
	}

	if _, inMem := getFromMemory(hashKey(target)); inMem {
		t.Fatal("target should have been LRU-evicted from memory")
	}
	if _, onDisk := getFromFile(hashKey(target), 600); !onDisk {
		t.Fatal("target's file copy should have survived memory eviction")
	}

	// Index entry must still exist so DeleteByPrefix can find the file copy.
	rawKeyMu.RLock()
	_, indexed := rawKeyIndex[target]
	rawKeyMu.RUnlock()
	if !indexed {
		t.Fatal("layered entry's raw key should NOT be dropped on memory eviction")
	}

	DeleteByPrefix("https://server/library/metadata/")
	if _, onDisk := getFromFile(hashKey(target), 600); onDisk {
		t.Error("DeleteByPrefix should have deleted the file copy")
	}
}

func TestDeleteByPrefix_NoMatchIsNoop(t *testing.T) {
	SetFileCacheDir(t.TempDir())
	Clear() //nolint:errcheck

	_ = Store("keep-me", []byte("v"), Layered, 60)

	DeleteByPrefix("nonexistent-prefix")

	if _, ok := Get("keep-me", Layered, 60); !ok {
		t.Error("DeleteByPrefix with non-matching prefix should leave entries intact")
	}
}
