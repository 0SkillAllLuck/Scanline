package hubs

import "context"

// ContinueWatching returns the continue watching hub.
//
// This contains items the user has started but not finished watching.
func (h *Hubs) ContinueWatching(ctx context.Context) ([]Hub, error) {
	var resp mediaContainerResponse[hubsContainer]
	err := h.GetMemCachedLibraries(ctx, "/hubs/continueWatching", 60).
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Hub, nil
}
