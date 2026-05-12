package cacheutils

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestFile_BinaryFormat verifies that storeInFile lays out the documented
// header (SCL1 magic + version + reserved + expiry) followed by raw,
// unmodified payload bytes — no base64, no JSON wrapper.
func TestFile_BinaryFormat(t *testing.T) {
	SetFileCacheDir(t.TempDir())
	Clear() //nolint:errcheck

	key := "https://server/photo/abc"
	// Use bytes that would be base64-mangled by the previous JSON format.
	payload := []byte{0x00, 0xFF, 0x89, 'S', 'C', 'L', '1', 0xAB}

	if err := Store(key, payload, Layered, 0); err != nil {
		t.Fatalf("Store: %v", err)
	}

	raw, err := os.ReadFile(cacheFilePath(hashKey(key)))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if len(raw) != cacheHeaderSize+len(payload) {
		t.Fatalf("on-disk size %d, want header (%d) + payload (%d)", len(raw), cacheHeaderSize, len(payload))
	}
	if string(raw[0:4]) != cacheFileMagic {
		t.Errorf("magic = %q, want %q", raw[0:4], cacheFileMagic)
	}
	if raw[4] != cacheFileVersion {
		t.Errorf("version = %d, want %d", raw[4], cacheFileVersion)
	}
	// reserved bytes zero
	for i := 5; i < 8; i++ {
		if raw[i] != 0 {
			t.Errorf("reserved byte %d = %d, want 0", i, raw[i])
		}
	}
	// ttl=0 → embedded expiry is zero
	if got := int64(binary.LittleEndian.Uint64(raw[8:16])); got != 0 {
		t.Errorf("expiresAt for ttl=0 = %d, want 0", got)
	}
	// Payload byte-for-byte identical to input
	if string(raw[cacheHeaderSize:]) != string(payload) {
		t.Errorf("payload tampered: got %v, want %v", raw[cacheHeaderSize:], payload)
	}
}

// TestFile_TtlEncodedAndEnforced ensures non-zero ttl writes the expiry
// into the header and that reads honour it.
func TestFile_TtlEncodedAndEnforced(t *testing.T) {
	SetFileCacheDir(t.TempDir())
	Clear() //nolint:errcheck

	key := "ttl-key"
	if err := Store(key, []byte("payload"), Layered, 1); err != nil {
		t.Fatalf("Store: %v", err)
	}

	raw, err := os.ReadFile(cacheFilePath(hashKey(key)))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	expiresAt := time.Unix(0, int64(binary.LittleEndian.Uint64(raw[8:16])))
	if expiresAt.Before(time.Now()) || expiresAt.After(time.Now().Add(2*time.Second)) {
		t.Errorf("expiresAt %v not within (now, now+2s)", expiresAt)
	}

	// Immediate read with positive ttl: hit.
	if _, ok := Get(key, Layered, 1); !ok {
		t.Fatal("expected fresh entry to be readable")
	}

	// Wait past expiry, then read with positive ttl: miss, and file gone.
	time.Sleep(1100 * time.Millisecond)
	clearMemory() // bypass the memory layer to force a file read
	if _, ok := Get(key, Layered, 1); ok {
		t.Error("expired entry should not be returned when ttl > 0")
	}
	if _, err := os.Stat(cacheFilePath(hashKey(key))); !os.IsNotExist(err) {
		t.Errorf("expired file should have been removed, stat err = %v", err)
	}
}

// TestFile_LegacyEntriesIgnored simulates migration from the old JSON-
// wrapped .json files. Pre-existing legacy files must not be misread as
// the new format, and Clear() must still remove them so the cache dir
// stays tidy.
func TestFile_LegacyEntriesIgnored(t *testing.T) {
	dir := t.TempDir()
	SetFileCacheDir(dir)
	Clear() //nolint:errcheck

	// Drop an artefact resembling the old JSON wrapper.
	legacyPath := filepath.Join(dir, "deadbeef.json")
	if err := os.WriteFile(legacyPath, []byte(`{"data":"AAAA","expires_at":"2030-01-01T00:00:00Z"}`), 0644); err != nil {
		t.Fatalf("seed legacy file: %v", err)
	}

	// A Get against any key should miss (we don't read .json any more) and
	// must not crash on the legacy artefact.
	if _, ok := Get("anything", Layered, 0); ok {
		t.Error("Get on unrelated key returned a hit")
	}

	// Legacy file still on disk (we don't proactively scan-and-delete).
	if _, err := os.Stat(legacyPath); err != nil {
		t.Errorf("legacy file should still exist until Clear(): %v", err)
	}

	// Clear() wipes every non-dir file regardless of extension.
	if err := Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Errorf("legacy file should be removed after Clear, stat err = %v", err)
	}
}

// TestFile_CorruptHeaderDropped verifies that a file with bad magic is
// removed rather than being returned as a "hit".
func TestFile_CorruptHeaderDropped(t *testing.T) {
	SetFileCacheDir(t.TempDir())
	Clear() //nolint:errcheck

	hashed := hashKey("corrupt-key")
	corrupt := make([]byte, cacheHeaderSize+8)
	copy(corrupt[0:4], "XXXX") // bad magic
	if err := os.WriteFile(cacheFilePath(hashed), corrupt, 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, ok := getFromFile(hashed, 0); ok {
		t.Error("getFromFile returned a hit for a corrupt-magic file")
	}
	if _, err := os.Stat(cacheFilePath(hashed)); !os.IsNotExist(err) {
		t.Errorf("corrupt file should be removed on read, stat err = %v", err)
	}
}

// TestFile_AtomicWrite_NoLeftoverTemp ensures successful Store doesn't
// leave the staging temp file behind in the cache directory.
func TestFile_AtomicWrite_NoLeftoverTemp(t *testing.T) {
	dir := t.TempDir()
	SetFileCacheDir(dir)
	Clear() //nolint:errcheck

	if err := Store("k", []byte("payload"), Layered, 0); err != nil {
		t.Fatalf("Store: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "_tmp-") {
			t.Errorf("leftover staging file: %s", e.Name())
		}
	}
}

// TestFile_AtomicWrite_ConcurrentReadSeesAllOrNothing verifies that a
// reader running while a writer is mid-Store never observes a partial
// payload. Pre-atomicity, an os.WriteFile would O_TRUNC the file and a
// concurrent reader could read 16+ bytes (valid header, truncated data),
// poisoning the memory cache with a corrupt entry that looks legitimate.
func TestFile_AtomicWrite_ConcurrentReadSeesAllOrNothing(t *testing.T) {
	dir := t.TempDir()
	SetFileCacheDir(dir)
	Clear() //nolint:errcheck

	const key = "concurrent-key"
	oldPayload := bytes.Repeat([]byte{'O'}, 64*1024)
	newPayload := bytes.Repeat([]byte{'N'}, 64*1024)

	// Seed with the old payload so readers have something to find.
	if err := Store(key, oldPayload, Layered, 0); err != nil {
		t.Fatalf("seed Store: %v", err)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Reader: continuously bypasses the memory layer and reads from disk.
	// Each successful read must return either oldPayload or newPayload in
	// full — never a truncated mix.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			clearMemory()
			got, ok := Get(key, Layered, 0)
			if !ok {
				continue
			}
			if !bytes.Equal(got, oldPayload) && !bytes.Equal(got, newPayload) {
				t.Errorf("partial read: len=%d first=%q last=%q", len(got), got[:1], got[len(got)-1:])
				return
			}
		}
	}()

	// Writer: alternate between the two payloads to maximise the chance
	// of catching a concurrent partial read.
	for i := 0; i < 200; i++ {
		payload := oldPayload
		if i%2 == 1 {
			payload = newPayload
		}
		if err := Store(key, payload, Layered, 0); err != nil {
			t.Fatalf("Store iter %d: %v", i, err)
		}
	}

	close(stop)
	wg.Wait()
}
