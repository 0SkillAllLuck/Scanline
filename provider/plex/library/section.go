package library

import "context"

// Section returns information about a specific library section.
func (l *Library) Section(ctx context.Context, sectionID string) (*LibrarySection, error) {
	var resp mediaContainerResponse[LibrarySection]
	err := l.Get(ctx, "/library/sections/"+sectionID).
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return &resp.MediaContainer, nil
}
