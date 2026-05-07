package plex

import (
	"testing"

	"github.com/0skillallluck/scanline/utils/cacheutils"
)

// TestInvalidateAfterPlayback_RemovesItemAndHubs verifies that the prefix
// invalidation drops cached entries for the affected item, the parent's
// children listing, the home hub, and the continue-watching hub — and leaves
// unrelated entries intact.
func TestInvalidateAfterPlayback_RemovesItemAndHubs(t *testing.T) {
	cacheutils.SetFileCacheDir(t.TempDir())
	cacheutils.Clear() //nolint:errcheck

	const baseURL = "https://server"

	// Seed a variety of cache entries.
	entries := map[string]struct {
		strategy cacheutils.Strategy
	}{
		baseURL + "/library/metadata/42":                          {cacheutils.Layered},
		baseURL + "/library/metadata/42?includeMarkers=1":         {cacheutils.Layered},
		baseURL + "/library/metadata/parent/children":             {cacheutils.Layered},
		baseURL + "/library/metadata/grandparent/children":        {cacheutils.Layered},
		baseURL + "/library/metadata/99":                          {cacheutils.Layered}, // unrelated
		baseURL + "/library/sections/all":                         {cacheutils.Layered}, // unrelated
		baseURL + "/hubs/promoted":                                {cacheutils.Layered},
		baseURL + "/hubs/continueWatching":                        {cacheutils.MemoryOnly},
	}
	for k, v := range entries {
		_ = cacheutils.Store(k, []byte("v:"+k), v.strategy, 600)
	}

	c := NewClient(baseURL, "tok", "cid")
	c.InvalidateAfterPlayback("42", "parent", "grandparent")

	expectGone := []string{
		baseURL + "/library/metadata/42",
		baseURL + "/library/metadata/42?includeMarkers=1",
		baseURL + "/library/metadata/parent/children",
		baseURL + "/library/metadata/grandparent/children",
		baseURL + "/hubs/promoted",
		baseURL + "/hubs/continueWatching",
	}
	for _, k := range expectGone {
		if _, ok := cacheutils.Get(k, entries[k].strategy, 600); ok {
			t.Errorf("expected %q to be invalidated", k)
		}
	}

	expectKept := []string{
		baseURL + "/library/metadata/99",
		baseURL + "/library/sections/all",
	}
	for _, k := range expectKept {
		if _, ok := cacheutils.Get(k, entries[k].strategy, 600); !ok {
			t.Errorf("expected %q to remain cached", k)
		}
	}
}

// TestInvalidateAfterPlayback_EmptyParentsAreSkipped checks that empty parent
// IDs don't accidentally invalidate the entire children prefix.
func TestInvalidateAfterPlayback_EmptyParentsAreSkipped(t *testing.T) {
	cacheutils.SetFileCacheDir(t.TempDir())
	cacheutils.Clear() //nolint:errcheck

	const baseURL = "https://server"
	_ = cacheutils.Store(baseURL+"/library/metadata/parent/children", []byte("v"), cacheutils.Layered, 600)

	c := NewClient(baseURL, "tok", "cid")
	c.InvalidateAfterPlayback("42", "", "")

	// The unrelated /children entry must survive empty-parent invalidation.
	if _, ok := cacheutils.Get(baseURL+"/library/metadata/parent/children", cacheutils.Layered, 600); !ok {
		t.Error("empty parent ratingKey should not blow away unrelated children entries")
	}
}
