package cacheutils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

var fileCacheDir = ""

type fileCachedEntry struct {
	Data      []byte    `json:"data"`
	ExpiresAt time.Time `json:"expires_at"`
}

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

func getFromFile(hashedKey string, ttl int) ([]byte, bool) {
	cacheDir := getFileCacheDir()
	filePath := filepath.Join(cacheDir, hashedKey+".json")

	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false
	}

	var cached fileCachedEntry
	if err := json.Unmarshal(raw, &cached); err != nil {
		os.Remove(filePath)
		return nil, false
	}

	if ttl > 0 && time.Now().After(cached.ExpiresAt) {
		os.Remove(filePath)
		return nil, false
	}

	return cached.Data, true
}

func storeInFile(hashedKey string, data []byte, ttl int) error {
	cacheDir := getFileCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	var expiresAt time.Time
	if ttl == 0 {
		expiresAt = time.Now().AddDate(100, 0, 0)
	} else {
		expiresAt = time.Now().Add(time.Duration(ttl) * time.Second)
	}

	cached := fileCachedEntry{
		Data:      data,
		ExpiresAt: expiresAt,
	}

	raw, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	filePath := filepath.Join(cacheDir, hashedKey+".json")
	return os.WriteFile(filePath, raw, 0644)
}

func deleteFromFile(hashedKey string) {
	cacheDir := getFileCacheDir()
	filePath := filepath.Join(cacheDir, hashedKey+".json")
	os.Remove(filePath)
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
			os.Remove(filepath.Join(cacheDir, entry.Name()))
		}
	}

	return nil
}
