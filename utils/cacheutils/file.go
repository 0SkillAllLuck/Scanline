package cacheutils

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// File-cache binary format. A small fixed header is followed by raw data
// bytes — no base64 or JSON wrapping, so binary payloads are stored
// verbatim and expiry can be checked from the first 16 bytes alone.
//
//	offset  size  field
//	0       4     magic = 'S','C','L','1'         (Scanline cache v1)
//	4       1     version
//	5       3     reserved (zero)
//	8       8     expiresAtUnixNano int64 LE      (0 = no expiry)
//	16      N     data
//
// Files use the .cache extension. Any leftover .json artefacts from an
// older format are ignored on read and removed by Clear().
const (
	cacheFileMagic   = "SCL1"
	cacheFileVersion = 1
	cacheHeaderSize  = 16
	cacheExtension   = ".cache"
)

var (
	errCacheBadMagic   = errors.New("cacheutils: file does not start with expected magic")
	errCacheBadVersion = errors.New("cacheutils: unsupported file-cache version")
	errCacheTruncated  = errors.New("cacheutils: file shorter than header")
)

var fileCacheDir = ""

// SetFileCacheDir overrides the file cache directory (useful for tests).
func SetFileCacheDir(dir string) {
	fileCacheDir = dir
}

func getFileCacheDir() string {
	if fileCacheDir != "" {
		return fileCacheDir
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}

	fileCacheDir = filepath.Join(cacheDir, "scanline")
	if err := os.MkdirAll(fileCacheDir, 0755); err != nil {
		fileCacheDir = filepath.Join(os.TempDir(), "scanline-cache")
		os.MkdirAll(fileCacheDir, 0755) //nolint:errcheck // last-resort fallback
	}

	return fileCacheDir
}

func cacheFilePath(hashedKey string) string {
	return filepath.Join(getFileCacheDir(), hashedKey+cacheExtension)
}

// getFromFile reads a cache entry from disk. ttl > 0 enforces the embedded
// expiry; ttl == 0 returns the entry regardless of staleness.
//
// The returned slice aliases the on-disk bytes (no copy). Callers that
// retain it past the immediate call must copy; the Layered Get path is
// safe because storeInMemory copies via copyBytes before caching.
func getFromFile(hashedKey string, ttl int) ([]byte, bool) {
	filePath := cacheFilePath(hashedKey)

	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false
	}

	expiresAt, payload, err := parseCacheFile(raw)
	if err != nil {
		// Malformed / unknown-version file — drop it so the next write
		// can replace it cleanly.
		_ = os.Remove(filePath)
		return nil, false
	}

	if ttl > 0 && !expiresAt.IsZero() && time.Now().After(expiresAt) {
		_ = os.Remove(filePath)
		return nil, false
	}

	return payload, true
}

// storeInFile writes a cache entry atomically: the payload is staged in a
// sibling temp file and then renamed into place. Without this, a concurrent
// reader could observe a partial write whose header validates but whose
// payload is truncated, and that corrupt entry would keep returning as a
// "valid" hit until evicted. Skipping fsync is deliberate — for a cache,
// the read-side bad-magic check self-heals after a crash and the per-write
// flush cost would dominate the Store path.
func storeInFile(hashedKey string, data []byte, ttl int) error {
	cacheDir := getFileCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	var expiresAtNano int64
	if ttl > 0 {
		expiresAtNano = time.Now().Add(time.Duration(ttl) * time.Second).UnixNano()
	}

	buf := make([]byte, cacheHeaderSize+len(data))
	copy(buf[0:4], cacheFileMagic)
	buf[4] = cacheFileVersion
	// buf[5:8] is reserved and already zero from make().
	binary.LittleEndian.PutUint64(buf[8:16], uint64(expiresAtNano))
	copy(buf[cacheHeaderSize:], data)

	tmp, err := os.CreateTemp(cacheDir, "_tmp-*"+cacheExtension)
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(buf); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, cacheFilePath(hashedKey)); err != nil {
		return err
	}
	committed = true
	return nil
}

func deleteFromFile(hashedKey string) {
	_ = os.Remove(cacheFilePath(hashedKey))
}

// parseCacheFile validates the header and returns the embedded expiry plus
// the payload slice. The payload aliases raw — no copy is made.
func parseCacheFile(raw []byte) (time.Time, []byte, error) {
	if len(raw) < cacheHeaderSize {
		return time.Time{}, nil, errCacheTruncated
	}
	if string(raw[0:4]) != cacheFileMagic {
		return time.Time{}, nil, errCacheBadMagic
	}
	if raw[4] != cacheFileVersion {
		return time.Time{}, nil, errCacheBadVersion
	}
	expiresAtNano := int64(binary.LittleEndian.Uint64(raw[8:16]))
	var expiresAt time.Time
	if expiresAtNano != 0 {
		expiresAt = time.Unix(0, expiresAtNano)
	}
	return expiresAt, raw[cacheHeaderSize:], nil
}

func clearFileDir() error {
	cacheDir := getFileCacheDir()
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			_ = os.Remove(filepath.Join(cacheDir, entry.Name()))
		}
	}

	return nil
}
