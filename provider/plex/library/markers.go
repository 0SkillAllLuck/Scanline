package library

import "context"

// Markers retrieves chapter markers (credits, intros, etc.) for a media item.
//
// The id parameter is the rating key of the item. Markers are fetched via the
// metadata endpoint with includeMarkers=1; the dedicated /marker sub-resource
// returns 404 on most Plex Media Server versions.
func (l *Library) Markers(ctx context.Context, id string) ([]Marker, error) {
	var resp mediaContainerResponse[metadataContainer]
	err := l.GetWithQuery(ctx, "/library/metadata/"+id, map[string]string{
		"includeMarkers": "1",
	}).DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	if len(resp.MediaContainer.Metadata) == 0 {
		return nil, nil
	}
	return resp.MediaContainer.Metadata[0].Marker, nil
}
