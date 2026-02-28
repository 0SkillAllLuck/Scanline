package hubs

import "context"

// Section returns the hubs for a specific library section.
func (h *Hubs) Section(ctx context.Context, sectionID string) ([]Hub, error) {
	var resp mediaContainerResponse[hubsContainer]
	err := h.Get(ctx, "/hubs/sections/"+sectionID).
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Hub, nil
}
