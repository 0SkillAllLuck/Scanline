// Package library provides access to Plex library and metadata endpoints.
package library

import (
	"encoding/json"

	"github.com/0skillallluck/scanline/provider/plex/base"
)

// Library provides access to library and metadata endpoints.
type Library struct {
	*base.Base
}

// New creates a new Library service.
func New(b *base.Base) *Library {
	return &Library{Base: b}
}

// ContentOptions specifies options for content listing.
type ContentOptions struct {
	// Start is the starting offset for pagination.
	Start int

	// Size is the maximum number of items to return.
	Size int

	// Sort specifies the sort order (e.g., "titleSort", "addedAt:desc").
	Sort string

	// Type filters by content type (e.g., "1" for movies, "2" for shows).
	Type string
}

// LibrarySection represents a library section (e.g., Movies, TV Shows).
type LibrarySection struct {
	// Key is the section identifier used in API paths.
	Key string `json:"key"`

	// Type is the section type (movie, show, artist, photo).
	Type string `json:"type"`

	// Title is the display name of the section.
	Title string `json:"title"`

	// Agent is the metadata agent used for this section.
	Agent string `json:"agent"`

	// Scanner is the scanner used for this section.
	Scanner string `json:"scanner"`

	// Language is the preferred language for metadata.
	Language string `json:"language"`

	// UUID is the unique identifier for this section.
	UUID string `json:"uuid"`

	// UpdatedAt is the Unix timestamp of the last update.
	UpdatedAt int64 `json:"updatedAt,omitempty"`

	// ScannedAt is the Unix timestamp of the last scan.
	ScannedAt int64 `json:"scannedAt,omitempty"`

	// ContentChangedAt is the Unix timestamp of the last content change.
	ContentChangedAt int64 `json:"contentChangedAt,omitempty"`

	// Thumb is the URL path to the section thumbnail.
	Thumb string `json:"thumb,omitempty"`

	// Art is the URL path to the section artwork.
	Art string `json:"art,omitempty"`

	// Composite is the URL path to the section composite image.
	Composite string `json:"composite,omitempty"`
}

// Rating represents a rating from an external source.
type Rating struct {
	// Image is the rating source identifier (e.g., "rottentomatoes://image.rating.ripe").
	Image string `json:"image"`

	// Type is the rating type ("critic" or "audience").
	Type string `json:"type"`

	// Value is the rating value.
	Value float64 `json:"value"`
}

// Ratings is a slice of Rating that can unmarshal from either a number or an array.
type Ratings []Rating

// UnmarshalJSON implements custom unmarshaling for Ratings.
// It handles both a single number value and an array of Rating objects.
func (r *Ratings) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a slice first
	var ratings []Rating
	if err := json.Unmarshal(data, &ratings); err == nil {
		*r = ratings
		return nil
	}

	// Try to unmarshal as a single number (legacy format)
	var singleValue float64
	if err := json.Unmarshal(data, &singleValue); err == nil {
		// Create a single rating with the numeric value
		*r = []Rating{{Value: singleValue, Type: "critic"}}
		return nil
	}

	// If both fail, return empty slice
	*r = []Rating{}
	return nil
}

// Metadata represents a media item (movie, show, episode, album, track, etc.).
type Metadata struct {
	// RatingKey is the unique identifier for this item.
	RatingKey string `json:"ratingKey"`

	// Key is the API path to get full details.
	Key string `json:"key"`

	// Type is the item type (movie, show, season, episode, artist, album, track).
	Type string `json:"type"`

	// Title is the display title.
	Title string `json:"title"`

	// TitleSort is the title used for sorting.
	TitleSort string `json:"titleSort,omitempty"`

	// OriginalTitle is the original title (for foreign content).
	OriginalTitle string `json:"originalTitle,omitempty"`

	// Summary is the plot summary or description.
	Summary string `json:"summary,omitempty"`

	// Year is the release year.
	Year int `json:"year,omitempty"`

	// Thumb is the URL path to the poster/thumbnail.
	Thumb string `json:"thumb,omitempty"`

	// Art is the URL path to the background artwork.
	Art string `json:"art,omitempty"`

	// Duration is the runtime in milliseconds.
	Duration int `json:"duration,omitempty"`

	// ViewOffset is the playback position in milliseconds.
	ViewOffset int `json:"viewOffset,omitempty"`

	// ViewCount is the number of times this item has been watched.
	ViewCount int `json:"viewCount,omitempty"`

	// ChildCount is the number of direct children (seasons for shows, episodes for seasons).
	ChildCount int `json:"childCount,omitempty"`

	// LeafCount is the total number of leaf items (episodes for shows).
	LeafCount int `json:"leafCount,omitempty"`

	// AddedAt is the Unix timestamp when the item was added.
	AddedAt int64 `json:"addedAt,omitempty"`

	// UpdatedAt is the Unix timestamp when the item was last updated.
	UpdatedAt int64 `json:"updatedAt,omitempty"`

	// Index is the item's position (episode number, track number).
	Index int `json:"index,omitempty"`

	// ParentIndex is the parent's position (season number).
	ParentIndex int `json:"parentIndex,omitempty"`

	// ParentRatingKey is the rating key of the parent item.
	ParentRatingKey string `json:"parentRatingKey,omitempty"`

	// ParentTitle is the title of the parent item.
	ParentTitle string `json:"parentTitle,omitempty"`

	// ParentThumb is the URL path to the parent's thumbnail.
	ParentThumb string `json:"parentThumb,omitempty"`

	// GrandparentTitle is the title of the grandparent (show title for episodes).
	GrandparentTitle string `json:"grandparentTitle,omitempty"`

	// GrandparentThumb is the URL path to the grandparent's thumbnail.
	GrandparentThumb string `json:"grandparentThumb,omitempty"`

	// ContentRating is the content rating (PG-13, TV-MA, etc.).
	ContentRating string `json:"contentRating,omitempty"`

	// Ratings contains all ratings from various sources.
	Ratings Ratings `json:"rating,omitempty"`

	// Tagline is the promotional tagline.
	Tagline string `json:"tagline,omitempty"`

	// UserRating is the user's personal rating.
	UserRating float64 `json:"userRating,omitempty"`

	// AudienceRating is the audience rating from external sources (legacy field).
	AudienceRating float64 `json:"audienceRating,omitempty"`

	// RatingImage is the URL path to the rating source logo (legacy field).
	RatingImage string `json:"ratingImage,omitempty"`

	// AudienceRatingImage is the URL path to the audience rating source logo (legacy field).
	AudienceRatingImage string `json:"audienceRatingImage,omitempty"`

	// Studio is the production studio.
	Studio string `json:"studio,omitempty"`

	// OriginallyAvailableAt is the original release date (YYYY-MM-DD).
	OriginallyAvailableAt string `json:"originallyAvailableAt,omitempty"`

	// Media contains the available media versions (quality variants).
	Media []Media `json:"Media,omitempty"`

	// Genre contains the genre tags.
	Genre []Tag `json:"Genre,omitempty"`

	// Director contains the director credits.
	Director []Tag `json:"Director,omitempty"`

	// Writer contains the writer credits.
	Writer []Tag `json:"Writer,omitempty"`

	// Role contains the cast/actor credits.
	Role []Tag `json:"Role,omitempty"`
}

// Media represents a media version with specific encoding/quality.
type Media struct {
	// ID is the unique identifier for this media version.
	ID int `json:"id"`

	// Duration is the runtime in milliseconds.
	Duration int `json:"duration,omitempty"`

	// Bitrate is the overall bitrate in kbps.
	Bitrate int `json:"bitrate,omitempty"`

	// Width is the video width in pixels.
	Width int `json:"width,omitempty"`

	// Height is the video height in pixels.
	Height int `json:"height,omitempty"`

	// AspectRatio is the video aspect ratio.
	AspectRatio float64 `json:"aspectRatio,omitempty"`

	// AudioChannels is the number of audio channels.
	AudioChannels int `json:"audioChannels,omitempty"`

	// AudioCodec is the audio codec (aac, ac3, dts, etc.).
	AudioCodec string `json:"audioCodec,omitempty"`

	// VideoCodec is the video codec (h264, hevc, etc.).
	VideoCodec string `json:"videoCodec,omitempty"`

	// VideoResolution is the resolution label (1080, 720, 4k, etc.).
	VideoResolution string `json:"videoResolution,omitempty"`

	// Container is the file container format (mkv, mp4, etc.).
	Container string `json:"container,omitempty"`

	// VideoFrameRate is the frame rate category (24p, NTSC, PAL, etc.).
	VideoFrameRate string `json:"videoFrameRate,omitempty"`

	// VideoProfile is the video codec profile (main, high, etc.).
	VideoProfile string `json:"videoProfile,omitempty"`

	// Part contains the media file parts.
	Part []Part `json:"Part,omitempty"`
}

// Part represents a media file part (for multi-part files).
type Part struct {
	// ID is the unique identifier for this part.
	ID int `json:"id"`

	// Key is the API path to access this part.
	Key string `json:"key"`

	// Duration is the part duration in milliseconds.
	Duration int `json:"duration,omitempty"`

	// File is the filesystem path to the file.
	File string `json:"file,omitempty"`

	// Size is the file size in bytes.
	Size int64 `json:"size,omitempty"`

	// Container is the file container format.
	Container string `json:"container,omitempty"`

	// VideoProfile is the video codec profile.
	VideoProfile string `json:"videoProfile,omitempty"`

	// Stream contains the individual media streams.
	Stream []Stream `json:"Stream,omitempty"`
}

// Stream represents an individual media stream (video, audio, subtitle).
type Stream struct {
	// ID is the unique identifier for this stream.
	ID int `json:"id"`

	// StreamType indicates the stream type (1=video, 2=audio, 3=subtitle).
	StreamType int `json:"streamType"`

	// Codec is the codec name.
	Codec string `json:"codec,omitempty"`

	// Index is the stream index in the container.
	Index int `json:"index,omitempty"`

	// Bitrate is the stream bitrate in kbps.
	Bitrate int `json:"bitrate,omitempty"`

	// Language is the human-readable language name.
	Language string `json:"language,omitempty"`

	// LanguageCode is the ISO 639-1 language code.
	LanguageCode string `json:"languageCode,omitempty"`

	// LanguageTag is the BCP 47 language tag.
	LanguageTag string `json:"languageTag,omitempty"`

	// Title is the stream title/name.
	Title string `json:"title,omitempty"`

	// DisplayTitle is the formatted display title.
	DisplayTitle string `json:"displayTitle,omitempty"`

	// Channels is the number of audio channels.
	Channels int `json:"channels,omitempty"`

	// SamplingRate is the audio sampling rate in Hz.
	SamplingRate int `json:"samplingRate,omitempty"`

	// BitDepth is the audio bit depth.
	BitDepth int `json:"bitDepth,omitempty"`

	// Width is the video width in pixels.
	Width int `json:"width,omitempty"`

	// Height is the video height in pixels.
	Height int `json:"height,omitempty"`

	// FrameRate is the video frame rate.
	FrameRate float64 `json:"frameRate,omitempty"`

	// ChromaSubsampling is the chroma subsampling format.
	ChromaSubsampling string `json:"chromaSubsampling,omitempty"`

	// Selected indicates if this stream is currently selected.
	Selected bool `json:"selected,omitempty"`

	// Default indicates if this is the default stream.
	Default bool `json:"default,omitempty"`

	// Forced indicates if this is a forced subtitle stream.
	Forced bool `json:"forced,omitempty"`
}

// Tag represents a metadata tag (genre, director, actor, etc.).
type Tag struct {
	// ID is the unique identifier for this tag.
	ID int `json:"id,omitempty"`

	// Tag is the tag text/name.
	Tag string `json:"tag"`

	// Count is the number of items with this tag.
	Count int `json:"count,omitempty"`

	// Role is the character name (for actor credits).
	Role string `json:"role,omitempty"`

	// Thumb is the URL path to the tag thumbnail (for actors).
	Thumb string `json:"thumb,omitempty"`
}

// Container types for JSON unmarshaling.

type mediaContainer struct {
	Size      int `json:"size"`
	TotalSize int `json:"totalSize,omitempty"`
	Offset    int `json:"offset,omitempty"`
}

type librarySectionsContainer struct {
	Directory []LibrarySection `json:"Directory"`
}

type metadataContainer struct {
	mediaContainer
	Metadata []Metadata `json:"Metadata"`
}

type mediaContainerResponse[T any] struct {
	MediaContainer T `json:"MediaContainer"`
}
