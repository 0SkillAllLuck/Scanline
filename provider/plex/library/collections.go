package library

import "context"

// Collections returns all collections in a library section.
func (l *Library) Collections(ctx context.Context, sectionID string) ([]Metadata, error) {
	var resp mediaContainerResponse[metadataContainer]
	query := map[string]string{"sectionID": sectionID}
	err := l.GetWithQuery(ctx, "/library/collections", query).
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Metadata, nil
}
