package library

import (
	"context"
	"strconv"
)

// Content returns items from a library section with optional pagination and filtering.
//
// Returns the items and the total count available (for pagination).
func (l *Library) Content(ctx context.Context, sectionID string, opts *ContentOptions) ([]Metadata, int, error) {
	query := make(map[string]string)
	if opts != nil {
		if opts.Start > 0 {
			query["X-Plex-Container-Start"] = strconv.Itoa(opts.Start)
		}
		if opts.Size > 0 {
			query["X-Plex-Container-Size"] = strconv.Itoa(opts.Size)
		}
		if opts.Sort != "" {
			query["sort"] = opts.Sort
		}
		if opts.Type != "" {
			query["type"] = opts.Type
		}
	}

	var resp mediaContainerResponse[metadataContainer]
	err := l.GetWithQuery(ctx, "/library/sections/"+sectionID+"/all", query).
		DoAndDecode(&resp)
	if err != nil {
		return nil, 0, err
	}
	return resp.MediaContainer.Metadata, resp.MediaContainer.TotalSize, nil
}
