package plex

import (
	"github.com/0skillallluck/scanline/provider/plex/hubs"
	"github.com/0skillallluck/scanline/provider/plex/library"
	"github.com/0skillallluck/scanline/provider/plex/playlists"
	"github.com/0skillallluck/scanline/provider/plex/search"
	"github.com/0skillallluck/scanline/provider/plex/server"
	"github.com/0skillallluck/scanline/provider/plex/timeline"
)

// Re-export types from subpackages for backward compatibility.

// Server types
type (
	ServerInfo     = server.ServerInfo
	ServerIdentity = server.ServerIdentity
)

// Library types
type (
	LibrarySection = library.LibrarySection
	Metadata       = library.Metadata
	Media          = library.Media
	Part           = library.Part
	Stream         = library.Stream
	Tag            = library.Tag
	ContentOptions = library.ContentOptions
)

// Hub types
type Hub = hubs.Hub

// SearchHub is the Hub type from the search sub-package.
type SearchHub = search.Hub

// Playlist types
type Playlist = playlists.Playlist

// PlaybackState is the PlaybackState type from the timeline sub-package.
type PlaybackState = timeline.PlaybackState

// Timeline state constants
const (
	StatePlaying = timeline.StatePlaying
	StatePaused  = timeline.StatePaused
	StateStopped = timeline.StateStopped
)
