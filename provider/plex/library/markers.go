package library

import "context"

// Markers retrieves chapter markers (credits, intros, etc.) for a media item.
//
// The id parameter is the rating key of the item.
func (l *Library) Markers(ctx context.Context, id string) ([]Marker, error) {
	var resp mediaContainerResponse[markerContainer]
	err := l.Get(ctx, "/library/metadata/"+id+"/marker").
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Marker, nil
}
