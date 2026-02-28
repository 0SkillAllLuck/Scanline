package playlists

import (
	"context"

	"github.com/0skillallluck/scanline/provider/plex/library"
)

// Items returns the items in a playlist.
func (p *Playlists) Items(ctx context.Context, playlistID string) ([]library.Metadata, error) {
	var resp mediaContainerResponse[metadataContainer]
	err := p.Get(ctx, "/playlists/"+playlistID+"/items").
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Metadata, nil
}
