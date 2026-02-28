package hubs

import "context"

// Related returns content related to a specific item.
//
// The id parameter is the rating key of the item.
func (h *Hubs) Related(ctx context.Context, id string) ([]Hub, error) {
	var resp mediaContainerResponse[hubsContainer]
	err := h.Get(ctx, "/hubs/metadata/"+id+"/related").
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Hub, nil
}
