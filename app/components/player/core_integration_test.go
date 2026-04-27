//go:build integration

// Integration tests for the playbin3 core wrapper. Excluded from the default
// `go test ./...` because they require:
//   - GStreamer base/good plugins
//   - gst-plugins-rs (gtk4paintablesink)
//   - A GTK-capable environment (a display, or Xvfb/GDK_BACKEND=wayland with
//     a compositor; gtk4paintablesink installs into the GTK widget tree even
//     for the synthetic test source).
//   - SCANLINE_INTEGRATION_VIDEO set to a file path or URI of a short test
//     video. playbin3 has no built-in URI handler for synthetic sources, so
//     the test cannot generate its own input.
//
// Run with: SCANLINE_INTEGRATION_VIDEO=/path/to/sample.mp4 \
//           go test -tags=integration ./app/components/player/

package player

import (
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-gst/go-gst/gst"
)

// TestCoreLifecycle exercises the construction → SetURI → Pause → Play → Close
// path. The point is to catch the common regressions: paintable double-unref
// on Close, bus watch never firing, state changes not reaching OnStateChange.
// It is NOT a substitute for end-to-end testing with a real Plex source —
// those still require manual QA.
func TestCoreLifecycle(t *testing.T) {
	uri := testVideoURI(t)

	var gotPaused, gotPlaying atomic.Bool
	stateCh := make(chan gst.State, 8)

	c, err := newCore(coreOptions{
		OnStateChange: func(s gst.State) {
			if s == gst.StatePaused {
				gotPaused.Store(true)
			}
			if s == gst.StatePlaying {
				gotPlaying.Store(true)
			}
			select {
			case stateCh <- s:
			default:
			}
		},
		OnError: func(err error) {
			t.Errorf("pipeline error: %v", err)
		},
	})
	if err != nil {
		t.Skipf("newCore failed (likely missing gst-plugins-rs): %v", err)
	}
	defer c.Close()

	c.SetURI(uri, 0)

	deadline := time.After(10 * time.Second)
	for !gotPaused.Load() {
		select {
		case <-stateCh:
		case <-deadline:
			t.Fatal("never reached PAUSED state")
		}
	}

	c.Play()
	deadline = time.After(10 * time.Second)
	for !gotPlaying.Load() {
		select {
		case <-stateCh:
		case <-deadline:
			t.Fatal("never reached PLAYING state")
		}
	}

	c.Pause()

	// Close is the bit we most want covered — its ordering invariants are
	// hand-rolled and easy to break (paintable unref before NULL state, bus
	// watch teardown, etc.). It must be safe to call twice.
	c.Close()
	c.Close()
}

// testVideoURI returns the URI from SCANLINE_INTEGRATION_VIDEO, normalising
// bare file paths into file:// URIs. Skips the test if unset or if the
// referenced path doesn't exist.
func testVideoURI(t *testing.T) string {
	t.Helper()
	v := os.Getenv("SCANLINE_INTEGRATION_VIDEO")
	if v == "" {
		t.Skip("SCANLINE_INTEGRATION_VIDEO not set; skipping (set to a file path or URI of a short test video)")
	}
	if u, err := url.Parse(v); err == nil && u.Scheme != "" {
		return v
	}
	abs, err := filepath.Abs(v)
	if err != nil {
		t.Fatalf("could not resolve %q: %v", v, err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("SCANLINE_INTEGRATION_VIDEO %q: %v", abs, err)
	}
	return "file://" + abs
}
