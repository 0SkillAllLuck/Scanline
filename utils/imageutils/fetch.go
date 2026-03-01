package imageutils

import (
	"fmt"
	"io"
	"net/http"

	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/utils/cacheutils"
)

func fetch(url string) ([]byte, error) {
	if preference.Performance().ShouldCacheImages() {
		if data, ok := cacheutils.Get(url, cacheutils.Layered, 0); ok {
			return data, nil
		}
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if preference.Performance().ShouldCacheImages() {
		cacheutils.Store(url, data, cacheutils.Layered, 0) //nolint:errcheck // best-effort cache
	}

	return data, nil
}
