package plex

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/0skillallluck/scanline/provider/plex/base"
	"github.com/0skillallluck/scanline/provider/plex/hubs"
	"github.com/0skillallluck/scanline/provider/plex/library"
	"github.com/0skillallluck/scanline/provider/plex/playlists"
	"github.com/0skillallluck/scanline/provider/plex/search"
	"github.com/0skillallluck/scanline/provider/plex/server"
	"github.com/0skillallluck/scanline/provider/plex/timeline"
	"github.com/0skillallluck/scanline/utils/httputils/request"
)

// Client provides access to a Plex Media Server API.
//
// Create a new client using [NewClient]:
//
//	client := plex.NewClient("http://localhost:32400", "your-token", "client-id")
//
// API methods are grouped by their sub-service:
//
//	info, err := client.Server.Info(ctx)
//	content, total, err := client.Library.Content(ctx, "1", nil)
type Client struct {
	base *base.Base

	// Server provides access to server information endpoints.
	Server *server.Server

	// Library provides access to library and metadata endpoints.
	Library *library.Library

	// Hubs provides access to hub endpoints (home, continue watching, related).
	Hubs *hubs.Hubs

	// Search provides access to search endpoints.
	Search *search.Search

	// Playlists provides access to playlist management endpoints.
	Playlists *playlists.Playlists

	// Timeline provides access to playback progress and scrobbling endpoints.
	Timeline *timeline.Timeline
}

// NewClient creates a new Plex Media Server client.
//
// The serverURL should be the base URL of the Plex server (e.g., "http://localhost:32400").
// The token is the authentication token for the server.
// The clientID should be a unique identifier for the application instance (typically a UUID).
func NewClient(serverURL, token, clientID string) *Client {
	b := &base.Base{
		BaseURL:  serverURL,
		Token:    token,
		ClientID: clientID,
	}

	return &Client{
		base:      b,
		Server:    server.New(b),
		Library:   library.New(b),
		Hubs:      hubs.New(b),
		Search:    search.New(b),
		Playlists: playlists.New(b),
		Timeline:  timeline.New(b),
	}
}

// ServerURL returns the base URL of the Plex server.
func (c *Client) ServerURL() string {
	return c.base.BaseURL
}

// Token returns the authentication token.
func (c *Client) Token() string {
	return c.base.Token
}

// ClientID returns the client identifier.
func (c *Client) ClientID() string {
	return c.base.ClientID
}

// SetServerURL updates the server URL.
func (c *Client) SetServerURL(url string) {
	c.base.BaseURL = url
}

// SetToken updates the authentication token.
func (c *Client) SetToken(token string) {
	c.base.Token = token
}

// TranscodeParams describes transcode settings for a media item.
type TranscodeParams struct {
	// RatingKey is the unique identifier for the item.
	RatingKey string

	// SessionID is the unique session identifier for playback tracking.
	SessionID string

	// DirectStreamAudio indicates if audio should be direct streamed (not transcoded).
	DirectStreamAudio bool

	// MaxBitrate is the maximum video bitrate in kbps (0 for original quality).
	MaxBitrate int

	// MaxResolution is the maximum video resolution (e.g., "1920x1080").
	MaxResolution string

	// AudioStreamID is the ID of the audio stream to use.
	AudioStreamID int

	// SubtitleStreamID is the ID of the subtitle stream to burn in.
	SubtitleStreamID int

	// Offset is the playback start position in seconds (for seeking).
	Offset int
}

// clientProfileExtra is an XML snippet that tells the Plex server what
// codecs/containers this client supports.
const clientProfileExtra = `add-transcode-target(type=videoProfile&context=streaming&protocol=http&container=mkv&videoCodec=h264,hevc,vp9,av1&audioCodec=aac,ac3,eac3,flac,opus,vorbis)+add-transcode-target(type=videoProfile&context=streaming&protocol=http&container=mp4&videoCodec=h264,hevc&audioCodec=aac,ac3,eac3)`

// PhotoTranscodeURL returns a URL for transcoded cover art with the given dimensions.
//
// The thumb parameter is the thumbnail path from metadata.
// Width and height specify the desired dimensions in pixels.
func (c *Client) PhotoTranscodeURL(thumb string, width, height int) string {
	if thumb == "" {
		return ""
	}
	return c.base.BaseURL + "/photo/:/transcode?width=" + fmt.Sprint(width) +
		"&height=" + fmt.Sprint(height) +
		"&minSize=1&upscale=1&url=" + url.QueryEscape(thumb+"?X-Plex-Token="+c.base.Token) +
		"&X-Plex-Token=" + c.base.Token
}

// StreamURL returns a direct play URL for the given media part key.
//
// The partKey is the key from a Part in the Media array of a Metadata item.
func (c *Client) StreamURL(partKey string) string {
	return c.base.BaseURL + partKey + "?X-Plex-Token=" + c.base.Token
}

// BuildTranscodeQuery builds Plex universal transcode query parameters.
func (c *Client) BuildTranscodeQuery(params TranscodeParams) url.Values {
	q := url.Values{}
	q.Set("hasMDE", "1")
	q.Set("path", "/library/metadata/"+params.RatingKey)
	q.Set("mediaIndex", "0")
	q.Set("partIndex", "0")
	q.Set("protocol", "http")
	q.Set("fastSeek", "1")
	q.Set("location", "lan")
	q.Set("session", params.SessionID)
	q.Set("X-Plex-Client-Profile-Extra", clientProfileExtra)
	q.Set("X-Plex-Client-Profile-Name", "Chrome")

	q.Set("directPlay", "0")
	q.Set("directStream", "1")
	q.Set("directStreamVideo", "1")

	if params.DirectStreamAudio {
		q.Set("directStreamAudio", "1")
	} else {
		if params.MaxBitrate > 0 {
			q.Set("maxVideoBitrate", fmt.Sprint(params.MaxBitrate))
		}
		if params.MaxResolution != "" {
			q.Set("videoResolution", params.MaxResolution)
		}
	}

	if params.AudioStreamID > 0 {
		q.Set("audioStreamID", fmt.Sprint(params.AudioStreamID))
	}
	if params.SubtitleStreamID > 0 {
		q.Set("subtitleStreamID", fmt.Sprint(params.SubtitleStreamID))
		q.Set("subtitles", "burn")
	}

	if params.Offset > 0 {
		q.Set("offset", fmt.Sprint(params.Offset))
	}

	return q
}

// TranscodeStartURL returns the URL for starting a transcode stream.
//
// The query parameters should be built using BuildTranscodeQuery.
// This URL includes authentication and can be used directly by media players.
func (c *Client) TranscodeStartURL(q url.Values) string {
	return c.base.BaseURL + "/video/:/transcode/universal/start.mkv?" + q.Encode() +
		"&X-Plex-Token=" + url.QueryEscape(c.base.Token) +
		"&X-Plex-Client-Identifier=" + url.QueryEscape(c.base.ClientID)
}

// MakeTranscodeDecision calls the Plex transcode decision endpoint.
//
// This should be called before playback to set up the session.
func (c *Client) MakeTranscodeDecision(ctx context.Context, q url.Values) error {
	resp, err := request.NewRequest(http.MethodGet, c.base.BaseURL+"/video/:/transcode/universal/decision").
		WithContext(ctx).
		WithHeaders(map[string]string{
			"X-Plex-Token":             c.base.Token,
			"X-Plex-Client-Identifier": c.base.ClientID,
			"Accept":                   "application/json",
		}).
		WithLogging("X-Plex-Token").
		WithQueryValues(q).
		Do()
	if err != nil {
		return fmt.Errorf("executing decision request: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("decision returned %d: %s", resp.StatusCode, string(resp.Body))
	}

	return nil
}

// ResolvePlaybackURL calls the decision endpoint for session tracking,
// then returns the direct play URL.
//
// The direct play URL is used because it supports HTTP range requests
// which are needed for seeking.
func (c *Client) ResolvePlaybackURL(ctx context.Context, partKey, ratingKey, sessionID string) string {
	params := TranscodeParams{
		RatingKey:         ratingKey,
		SessionID:         sessionID,
		DirectStreamAudio: true,
	}
	q := c.BuildTranscodeQuery(params)
	if err := c.MakeTranscodeDecision(ctx, q); err != nil {
		slog.Warn("plex: decision failed", "error", err)
	}
	return c.StreamURL(partKey)
}
