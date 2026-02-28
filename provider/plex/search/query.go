package search

import (
	"context"
	"strconv"
)

// Query searches for content across all libraries.
//
// The query parameter is the search term.
// The limit parameter specifies the maximum number of results per content type (0 for no limit).
// Results are returned grouped by content type in separate hubs.
func (s *Search) Query(ctx context.Context, query string, limit int) ([]Hub, error) {
	params := map[string]string{"query": query}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}

	var resp mediaContainerResponse[hubsContainer]
	err := s.GetWithQuery(ctx, "/hubs/search", params).
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Hub, nil
}
