// Package textures provides a process-wide cache for gdk.Texture instances
// loaded from GResource paths.
//
// The same resource (e.g. "missing-album.svg") is used by every poster card
// in the UI; without caching, each card construction crosses the CGO boundary
// to decode the SVG anew. Profiling showed NewTextureFromResource at ~5% of
// navigation CPU. This cache makes the second-and-later loads free.
package textures

import (
	"sync"

	"codeberg.org/puregotk/puregotk/v4/gdk"
)

var (
	cacheMu sync.RWMutex
	cache   = make(map[string]*gdk.Texture)
)

// FromResource returns the gdk.Texture for the given GResource path, caching
// the result. The returned texture is reference-counted by GDK; callers must
// not Unref it (the cache owns the only Go-side reference, and GTK will keep
// its own references for any widget that uses the texture as a paintable).
//
// Texture-from-resource is safe to call from any thread because the underlying
// GResource lookup is read-only and the GDK texture is immutable after
// creation.
func FromResource(path string) *gdk.Texture {
	cacheMu.RLock()
	if t, ok := cache[path]; ok {
		cacheMu.RUnlock()
		return t
	}
	cacheMu.RUnlock()

	cacheMu.Lock()
	defer cacheMu.Unlock()
	if t, ok := cache[path]; ok {
		return t
	}

	t := gdk.NewTextureFromResource(path)
	cache[path] = t
	return t
}
