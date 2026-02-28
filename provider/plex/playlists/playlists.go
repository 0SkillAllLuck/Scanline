// Package playlists provides access to Plex playlist management endpoints.
package playlists

import (
	"fmt"

	"github.com/0skillallluck/scanline/provider/plex/base"
	"github.com/0skillallluck/scanline/provider/plex/library"
)

// Playlists provides access to playlist management endpoints.
type Playlists struct {
	*base.Base
}

// New creates a new Playlists service.
func New(b *base.Base) *Playlists {
	return &Playlists{Base: b}
}

// Playlist represents a user playlist.
type Playlist struct {
	// RatingKey is the unique identifier for this playlist.
	RatingKey string `json:"ratingKey"`

	// Key is the API path to get playlist details.
	Key string `json:"key"`

	// Type is always "playlist".
	Type string `json:"type"`

	// Title is the playlist title.
	Title string `json:"title"`

	// Summary is the playlist description.
	Summary string `json:"summary,omitempty"`

	// Smart indicates if this is a smart (auto-updating) playlist.
	Smart bool `json:"smart"`

	// PlaylistType is the type of content (video, audio, photo).
	PlaylistType string `json:"playlistType"`

	// Composite is the URL path to the playlist cover image.
	Composite string `json:"composite,omitempty"`

	// Duration is the total duration in milliseconds.
	Duration int `json:"duration,omitempty"`

	// LeafCount is the number of items in the playlist.
	LeafCount int `json:"leafCount,omitempty"`

	// AddedAt is the Unix timestamp when the playlist was created.
	AddedAt int64 `json:"addedAt,omitempty"`

	// UpdatedAt is the Unix timestamp when the playlist was last updated.
	UpdatedAt int64 `json:"updatedAt,omitempty"`
}

// EmptyResultError indicates that a query returned no results.
type EmptyResultError struct {
	Resource string
}

func (e *EmptyResultError) Error() string {
	return fmt.Sprintf("%s not found", e.Resource)
}

// Container types for JSON unmarshaling.

type mediaContainer struct {
	Size int `json:"size"`
}

type playlistsContainer struct {
	mediaContainer
	Metadata []Playlist `json:"Metadata"`
}

type playlistContainer struct {
	mediaContainer
	Metadata []Playlist `json:"Metadata"`
}

type metadataContainer struct {
	mediaContainer
	Metadata []library.Metadata `json:"Metadata"`
}

type mediaContainerResponse[T any] struct {
	MediaContainer T `json:"MediaContainer"`
}
