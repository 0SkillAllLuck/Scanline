package library

import "context"

// Children returns direct child items for a parent item.
//
// The id parameter is the rating key of the parent (e.g., season ID to get episodes).
func (l *Library) Children(ctx context.Context, id string) ([]Metadata, error) {
	var resp mediaContainerResponse[metadataContainer]
	err := l.GetCachedMetadata(ctx, "/library/metadata/"+id+"/children", 24*60*60).
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Metadata, nil
}
