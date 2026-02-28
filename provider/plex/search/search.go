// Package search provides access to Plex search endpoints.
package search

import (
	"github.com/0skillallluck/scanline/provider/plex/base"
	"github.com/0skillallluck/scanline/provider/plex/hubs"
)

// Search provides access to search endpoints.
type Search struct {
	*base.Base
}

// New creates a new Search service.
func New(b *base.Base) *Search {
	return &Search{Base: b}
}

// Hub is an alias to hubs.Hub for search results.
type Hub = hubs.Hub

// Container types for JSON unmarshaling.

type hubsContainer struct {
	Hub []Hub `json:"Hub"`
}

type mediaContainerResponse[T any] struct {
	MediaContainer T `json:"MediaContainer"`
}
