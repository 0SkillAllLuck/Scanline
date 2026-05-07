package library

import "context"

// Seasons returns the seasons for a TV show.
//
// The id parameter is the rating key of the show.
func (l *Library) Seasons(ctx context.Context, id string) ([]Metadata, error) {
	var resp mediaContainerResponse[metadataContainer]
	err := l.GetCachedMetadata(ctx, "/library/metadata/"+id+"/children", 24*60*60).
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Metadata, nil
}
