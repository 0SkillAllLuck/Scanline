package playlists

import "context"

// Delete deletes a playlist.
func (p *Playlists) Delete(ctx context.Context, playlistID string) error {
	_, err := p.Base.Delete(ctx, "/playlists/"+playlistID).Do()
	return err
}
