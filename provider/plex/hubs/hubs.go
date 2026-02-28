// Package hubs provides access to Plex hub endpoints.
package hubs

import (
	"github.com/0skillallluck/scanline/provider/plex/base"
	"github.com/0skillallluck/scanline/provider/plex/library"
)

// Hubs provides access to hub endpoints (home, continue watching, related).
type Hubs struct {
	*base.Base
}

// New creates a new Hubs service.
func New(b *base.Base) *Hubs {
	return &Hubs{Base: b}
}

// Hub represents a content hub (e.g., "Continue Watching", "Recently Added").
type Hub struct {
	// Key is the API path to get more items from this hub.
	Key string `json:"key,omitempty"`

	// Title is the display title of the hub.
	Title string `json:"title"`

	// Type is the content type in this hub.
	Type string `json:"type"`

	// HubIdentifier is the unique identifier for this hub type.
	HubIdentifier string `json:"hubIdentifier"`

	// Context provides additional context about the hub.
	Context string `json:"context,omitempty"`

	// Size is the number of items in this hub.
	Size int `json:"size"`

	// More indicates if there are more items available.
	More bool `json:"more"`

	// Style is the display style for the hub.
	Style string `json:"style,omitempty"`

	// HubKey is an alternative key for the hub.
	HubKey string `json:"hubKey,omitempty"`

	// Promoted indicates if this is a promoted hub.
	Promoted bool `json:"promoted,omitempty"`

	// Metadata contains the items in this hub.
	Metadata []library.Metadata `json:"Metadata,omitempty"`
}

// Container types for JSON unmarshaling.

type hubsContainer struct {
	Hub []Hub `json:"Hub"`
}

type mediaContainerResponse[T any] struct {
	MediaContainer T `json:"MediaContainer"`
}
