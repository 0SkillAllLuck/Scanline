package hubs

import (
	"context"
	"strconv"
)

// Home returns the hubs displayed on the home screen.
//
// This includes recently added items, continue watching, and other personalized content.
// The count parameter specifies the maximum number of items to return per hub (0 for server default).
func (h *Hubs) Home(ctx context.Context, count int) ([]Hub, error) {
	var resp mediaContainerResponse[hubsContainer]

	query := make(map[string]string)
	if count > 0 {
		query["count"] = strconv.Itoa(count)
	}

	err := h.GetWithQuery(ctx, "/hubs", query).
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Hub, nil
}
