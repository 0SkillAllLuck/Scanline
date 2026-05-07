package library

import "context"

// Sections returns all library sections.
func (l *Library) Sections(ctx context.Context) ([]LibrarySection, error) {
	var resp mediaContainerResponse[librarySectionsContainer]
	err := l.GetCachedLibraries(ctx, "/library/sections/all", 24*60*60).
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.MediaContainer.Directory, nil
}
