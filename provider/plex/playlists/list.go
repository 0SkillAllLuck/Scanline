package playlists

import "context"

// List returns all playlists.
//
// Note: Mutations (Create/Update/Delete) bypass the in-memory cache, but
// their effect on subsequent List calls only becomes visible after the
// short TTL — there's no automatic invalidation hook on the playlist
// mutation endpoints today.
func (p *Playlists) List(ctx context.Context) ([]Playlist, error) {
	var resp mediaContainerResponse[playlistsContainer]
	err := p.GetMemCachedLibraries(ctx, "/playlists", 60).
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Metadata, nil
}
