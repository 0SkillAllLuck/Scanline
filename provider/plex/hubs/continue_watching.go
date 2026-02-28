package hubs

import "context"

// ContinueWatching returns the continue watching hub.
//
// This contains items the user has started but not finished watching.
// No caching is applied as this data changes frequently.
func (h *Hubs) ContinueWatching(ctx context.Context) ([]Hub, error) {
	var resp mediaContainerResponse[hubsContainer]
	err := h.Get(ctx, "/hubs/continueWatching").
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Hub, nil
}
