package library

import "context"

// Seasons returns the seasons for a TV show.
//
// The id parameter is the rating key of the show.
func (l *Library) Seasons(ctx context.Context, id string) ([]Metadata, error) {
	var resp mediaContainerResponse[metadataContainer]
	err := l.Get(ctx, "/library/metadata/"+id+"/children").
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Metadata, nil
}
