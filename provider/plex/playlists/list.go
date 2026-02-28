package playlists

import "context"

// List returns all playlists.
func (p *Playlists) List(ctx context.Context) ([]Playlist, error) {
	var resp mediaContainerResponse[playlistsContainer]
	err := p.Get(ctx, "/playlists").
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Metadata, nil
}
