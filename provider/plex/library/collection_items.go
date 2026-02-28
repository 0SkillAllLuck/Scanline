package library

import "context"

// CollectionItems returns the items in a collection.
func (l *Library) CollectionItems(ctx context.Context, collectionID string) ([]Metadata, error) {
	var resp mediaContainerResponse[metadataContainer]
	err := l.Get(ctx, "/library/collections/"+collectionID+"/items").
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Metadata, nil
}
