package library

import "context"

// Sections returns all library sections.
func (l *Library) Sections(ctx context.Context) ([]LibrarySection, error) {
	var resp mediaContainerResponse[librarySectionsContainer]
	err := l.Get(ctx, "/library/sections/all").
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Directory, nil
}
