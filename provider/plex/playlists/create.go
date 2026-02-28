package playlists

import "context"

// Create creates a new playlist.
//
// The title parameter is the playlist name.
// The playlistType should be "video", "audio", or "photo".
// The uri parameter is an optional server:// URI to initialize the playlist with items.
func (p *Playlists) Create(ctx context.Context, title, playlistType string, uri string) (*Playlist, error) {
	query := map[string]string{
		"title": title,
		"type":  playlistType,
		"smart": "0",
	}
	if uri != "" {
		query["uri"] = uri
	}

	var resp mediaContainerResponse[playlistContainer]
	err := p.PostWithQuery(ctx, "/playlists", query).
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	if len(resp.MediaContainer.Metadata) == 0 {
		return nil, &EmptyResultError{Resource: "playlist"}
	}
	return &resp.MediaContainer.Metadata[0], nil
}
