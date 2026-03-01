package sources

import (
	"context"
	"net/url"

	"github.com/0skillallluck/scanline/provider/plex"
)

// PlexSource adapts a plex.Client to the Source interface.
type PlexSource struct {
	client   *plex.Client
	serverID string
	name     string
}

// NewPlexSource creates a Source that wraps a Plex Media Server client.
func NewPlexSource(serverID, name string, client *plex.Client) *PlexSource {
	return &PlexSource{
		serverID: serverID,
		name:     name,
		client:   client,
	}
}

// PlexClient returns the underlying plex.Client for provider-specific operations
// (e.g. transcoding, timeline). Avoid using this for general media access.
func (s *PlexSource) PlexClient() *plex.Client {
	return s.client
}

func (s *PlexSource) ID() string   { return s.serverID }
func (s *PlexSource) Name() string { return s.name }

func (s *PlexSource) LibrarySections(ctx context.Context) ([]LibrarySection, error) {
	return s.client.Library.Sections(ctx)
}

func (s *PlexSource) LibrarySection(ctx context.Context, sectionID string) (*LibrarySection, error) {
	return s.client.Library.Section(ctx, sectionID)
}

func (s *PlexSource) LibraryContent(ctx context.Context, sectionID string, opts *ContentOptions) ([]Metadata, int, error) {
	return s.client.Library.Content(ctx, sectionID, opts)
}

func (s *PlexSource) GetMetadata(ctx context.Context, key string) (*Metadata, error) {
	return s.client.Library.Metadata(ctx, key)
}

func (s *PlexSource) GetChildren(ctx context.Context, key string) ([]Metadata, error) {
	return s.client.Library.Children(ctx, key)
}

func (s *PlexSource) HomeHubs(ctx context.Context) ([]Hub, error) {
	return s.client.Hubs.Home(ctx)
}

func (s *PlexSource) RelatedHubs(ctx context.Context, key string) ([]Hub, error) {
	return s.client.Hubs.Related(ctx, key)
}

func (s *PlexSource) Search(ctx context.Context, query string, limit int) ([]Hub, error) {
	return s.client.Search.Query(ctx, query, limit)
}

func (s *PlexSource) PhotoTranscodeURL(path string, width, height int) string {
	return s.client.PhotoTranscodeURL(path, width, height)
}

func (s *PlexSource) StreamURL(partKey string) string {
	return s.client.StreamURL(partKey)
}

func (s *PlexSource) ResolvePlaybackURL(ctx context.Context, partKey, ratingKey, sessionID string) string {
	return s.client.ResolvePlaybackURL(ctx, partKey, ratingKey, sessionID)
}

func (s *PlexSource) BuildTranscodeQuery(params TranscodeParams) url.Values {
	return s.client.BuildTranscodeQuery(params)
}

func (s *PlexSource) MakeTranscodeDecision(ctx context.Context, q url.Values) error {
	return s.client.MakeTranscodeDecision(ctx, q)
}

func (s *PlexSource) TranscodeStartURL(q url.Values) string {
	return s.client.TranscodeStartURL(q)
}

func (s *PlexSource) Scrobble(ctx context.Context, ratingKey string) error {
	return s.client.Timeline.Scrobble(ctx, ratingKey)
}

func (s *PlexSource) Unscrobble(ctx context.Context, ratingKey string) error {
	return s.client.Timeline.Unscrobble(ctx, ratingKey)
}

func (s *PlexSource) UpdateProgress(ctx context.Context, ratingKey string, state PlaybackState, timeMs, durationMs int) error {
	return s.client.Timeline.UpdateProgress(ctx, ratingKey, state, timeMs, durationMs)
}
