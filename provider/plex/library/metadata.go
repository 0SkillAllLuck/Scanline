package library

import (
	"context"
	"fmt"
)

// EmptyResultError indicates that a query returned no results.
type EmptyResultError struct {
	Resource string
	ID       string
}

func (e *EmptyResultError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// Metadata retrieves detailed information about a specific media item.
//
// The id parameter is the rating key of the item to retrieve.
// Returns EmptyResultError if the item is not found.
func (l *Library) Metadata(ctx context.Context, id string) (*Metadata, error) {
	var resp mediaContainerResponse[metadataContainer]
	err := l.Get(ctx, "/library/metadata/"+id).
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	if len(resp.MediaContainer.Metadata) == 0 {
		return nil, &EmptyResultError{Resource: "metadata", ID: id}
	}
	return &resp.MediaContainer.Metadata[0], nil
}
