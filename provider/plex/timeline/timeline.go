// Package timeline provides access to Plex playback progress and scrobbling endpoints.
package timeline

import "github.com/0skillallluck/scanline/provider/plex/base"

// Timeline provides access to playback progress and scrobbling endpoints.
type Timeline struct {
	*base.Base
}

// New creates a new Timeline service.
func New(b *base.Base) *Timeline {
	return &Timeline{Base: b}
}

// PlaybackState represents the current state of playback.
type PlaybackState string

// Playback states for timeline updates.
const (
	StatePlaying = PlaybackState("playing")
	StatePaused  = PlaybackState("paused")
	StateStopped = PlaybackState("stopped")
)
