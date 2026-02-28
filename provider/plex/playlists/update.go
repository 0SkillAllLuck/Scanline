package playlists

import "context"

// Update updates a playlist's title and/or summary.
//
// Pass empty strings to leave fields unchanged.
func (p *Playlists) Update(ctx context.Context, playlistID, title, summary string) error {
	query := make(map[string]string)
	if title != "" {
		query["title"] = title
	}
	if summary != "" {
		query["summary"] = summary
	}
	_, err := p.PutWithQuery(ctx, "/playlists/"+playlistID, query).Do()
	return err
}
