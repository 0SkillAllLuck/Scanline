package watchlist

import (
	"context"
	"fmt"
	"net/http"

	"github.com/0skillallluck/scanline/utils/httputils/request"
)

const discoverBaseURL = "https://discover.provider.plex.tv"

// Item represents an item from the Plex watchlist (online metadata from discover API).
type Item struct {
	RatingKey string `json:"ratingKey"`
	Key       string `json:"key"`
	Type      string `json:"type"` // "movie" or "show"
	Title     string `json:"title"`
	Year      int    `json:"year,omitempty"`
	Thumb     string `json:"thumb,omitempty"`
	Art       string `json:"art,omitempty"`
	Summary   string `json:"summary,omitempty"`
	GUID      string `json:"guid,omitempty"` // e.g. "plex://movie/..."
}

// Client fetches watchlist data from the Plex Discover API.
type Client struct {
	token    string
	clientID string
}

// NewClient creates a new watchlist client with account-level credentials.
func NewClient(token, clientID string) *Client {
	return &Client{
		token:    token,
		clientID: clientID,
	}
}

// watchlistResponse is the JSON envelope from the discover API.
type watchlistResponse struct {
	MediaContainer struct {
		Metadata []Item `json:"Metadata"`
	} `json:"MediaContainer"`
}

// List fetches the watchlist with the given filter ("all", "available", "released").
func (c *Client) List(ctx context.Context, filter string) ([]Item, error) {
	if filter == "" {
		filter = "all"
	}

	resp, err := request.NewRequest(http.MethodGet, discoverBaseURL+"/library/sections/watchlist/"+filter).
		WithContext(ctx).
		WithHeaders(map[string]string{
			"Accept":                   "application/json",
			"X-Plex-Token":             c.token,
			"X-Plex-Client-Identifier": c.clientID,
		}).
		WithLogging("X-Plex-Token").
		WithQuery(map[string]string{
			"includeCollections":   "1",
			"includeExternalMedia": "1",
		}).
		WithInMemoryCaching(5 * 60).
		WithCacheKey(c.token).
		Do()
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("watchlist request failed: %s", resp.Status)
	}

	var result watchlistResponse
	if err := resp.JSON(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return result.MediaContainer.Metadata, nil
}
