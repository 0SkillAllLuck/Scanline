package hubs

import "context"

// Home returns the hubs displayed on the home screen.
//
// This includes recently added items, continue watching, and other personalized content.
func (h *Hubs) Home(ctx context.Context) ([]Hub, error) {
	var resp mediaContainerResponse[hubsContainer]
	err := h.Get(ctx, "/hubs").
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Hub, nil
}
