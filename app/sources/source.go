package sources

import (
	"context"
	"net/url"
)

// Source provides provider-agnostic access to a media server.
type Source interface {
	// ID returns a unique identifier for this source (server ID).
	ID() string

	// Name returns the display name for this source.
	Name() string

	// LibrarySections returns all library sections available on this source.
	LibrarySections(ctx context.Context) ([]LibrarySection, error)

	// LibrarySection returns a specific library section by ID.
	LibrarySection(ctx context.Context, sectionID string) (*LibrarySection, error)

	// LibraryContent returns items from a library section with optional pagination.
	LibraryContent(ctx context.Context, sectionID string, opts *ContentOptions) ([]Metadata, int, error)

	// GetMetadata returns detailed information about a specific media item.
	GetMetadata(ctx context.Context, key string) (*Metadata, error)

	// GetChildren returns direct child items (show→seasons, season→episodes).
	GetChildren(ctx context.Context, key string) ([]Metadata, error)

	// HomeHubs returns the hubs displayed on the home screen.
	HomeHubs(ctx context.Context) ([]Hub, error)

	// RelatedHubs returns content related to a specific item.
	RelatedHubs(ctx context.Context, key string) ([]Hub, error)

	// Search queries the source for matching content.
	Search(ctx context.Context, query string, limit int) ([]Hub, error)

	// PhotoTranscodeURL returns a URL for transcoded cover art.
	PhotoTranscodeURL(path string, width, height int) string

	// StreamURL returns a direct play URL for the given media part key.
	StreamURL(partKey string) string

	// ResolvePlaybackURL resolves a playback URL via the decision endpoint.
	ResolvePlaybackURL(ctx context.Context, partKey, ratingKey, sessionID string) string

	// BuildTranscodeQuery builds transcode query parameters from the given params.
	BuildTranscodeQuery(params TranscodeParams) url.Values

	// MakeTranscodeDecision calls the transcode decision endpoint to set up a session.
	MakeTranscodeDecision(ctx context.Context, q url.Values) error

	// TranscodeStartURL returns the URL for starting a transcode stream.
	TranscodeStartURL(q url.Values) string

	// Scrobble marks an item as watched.
	Scrobble(ctx context.Context, ratingKey string) error

	// UpdateProgress reports playback position to the server.
	UpdateProgress(ctx context.Context, ratingKey string, state PlaybackState, timeMs, durationMs int) error
}
