// Package sources provides a provider-agnostic abstraction layer for media servers.
// UI code should use these types instead of importing provider-specific packages directly.
package sources

import (
	"github.com/0skillallluck/scanline/provider/plex"
	"github.com/0skillallluck/scanline/provider/plex/hubs"
	"github.com/0skillallluck/scanline/provider/plex/library"
	"github.com/0skillallluck/scanline/provider/plex/timeline"
)

// Type aliases for provider-agnostic access to media types.
// These decouple UI code from specific provider packages.
//
// NOTE: These are intentionally Plex-specific aliases for now.
// When adding a second provider, define provider-agnostic domain
// types here and have each provider adapter map to them.

type Metadata = library.Metadata
type Media = library.Media
type Part = library.Part
type Stream = library.Stream

const (
	StreamTypeVideo    = library.StreamTypeVideo
	StreamTypeAudio    = library.StreamTypeAudio
	StreamTypeSubtitle = library.StreamTypeSubtitle
)
type Tag = library.Tag
type Rating = library.Rating
type Ratings = library.Ratings
type LibrarySection = library.LibrarySection
type ContentOptions = library.ContentOptions
type Marker = library.Marker

const (
	MarkerTypeCredits = library.MarkerTypeCredits
	MarkerTypeIntro   = library.MarkerTypeIntro
)
type Hub = hubs.Hub
type TranscodeParams = plex.TranscodeParams
type PlaybackState = timeline.PlaybackState

// StreamSelection describes an explicit audio/subtitle selection for a media part.
// Nil fields mean "leave that stream type unchanged".
type StreamSelection struct {
	AudioStreamID    *int
	SubtitleStreamID *int
}

const (
	StatePlaying = timeline.StatePlaying
	StatePaused  = timeline.StatePaused
	StateStopped = timeline.StateStopped
)

// ArtURL returns the best art URL for a metadata item, with fallbacks
// for types where the primary art may be missing (e.g. episodes falling
// back to show poster).
func ArtURL(meta *Metadata) string {
	if meta.Art != "" {
		return meta.Art
	}
	if meta.Type == "episode" && meta.GrandparentThumb != "" {
		return meta.GrandparentThumb
	}
	return ""
}
