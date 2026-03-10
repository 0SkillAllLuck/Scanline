package watchlist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	Guid      string `json:"guid,omitempty"` // e.g. "plex://movie/..."
}

// Client fetches watchlist data from the Plex Discover API.
type Client struct {
	token    string
	clientID string
	http     *http.Client
}

// NewClient creates a new watchlist client with account-level credentials.
func NewClient(token, clientID string) *Client {
	return &Client{
		token:    token,
		clientID: clientID,
		http:     http.DefaultClient,
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

	url := discoverBaseURL + "/library/sections/watchlist/" + filter
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Token", c.token)
	req.Header.Set("X-Plex-Client-Identifier", c.clientID)

	q := req.URL.Query()
	q.Set("includeCollections", "1")
	q.Set("includeExternalMedia", "1")
	req.URL.RawQuery = q.Encode()

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("watchlist request failed: %s", resp.Status)
	}

	var result watchlistResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return result.MediaContainer.Metadata, nil
}
