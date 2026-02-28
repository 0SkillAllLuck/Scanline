// Package plex provides a client for interacting with Plex Media Server APIs.
//
// The package is organized around a central [Client] struct that provides access
// to various API endpoints through sub-services:
//
//   - [server.Server]: Server information and identity
//   - [library.Library]: Library sections, metadata, and collections
//   - [hubs.Hubs]: Home hubs, continue watching, and related content
//   - [search.Search]: Search functionality
//   - [playlists.Playlists]: Playlist management
//   - [timeline.Timeline]: Playback progress and scrobbling
//
// # Creating a Client
//
// Use [NewClient] to create a new client with a server URL and authentication token:
//
//	client := plex.NewClient("http://localhost:32400", "your-token", "client-id")
//
// # Making API Calls
//
// API methods are grouped by their sub-service:
//
//	// Get server information
//	info, err := client.Server.Info(ctx)
//
//	// Browse a library section
//	content, total, err := client.Library.Content(ctx, "1", nil)
//
//	// Search for content
//	results, err := client.Search.Query(ctx, "movie name", 10)
//
// # Error Handling
//
// The package provides typed errors for common API error conditions:
//
//   - [AuthenticationError]: 401 Unauthorized responses
//   - [AuthorizationError]: 403 Forbidden responses
//   - [NotFoundError]: 404 Not Found responses
//   - [ServerError]: 5xx server errors
//   - [EmptyResultError]: Query returned no results
//
// Use [errors.Is] to check for specific error types:
//
//	if errors.Is(err, plex.ErrNotFound) {
//	    // Handle not found
//	}
//
// # Authentication
//
// For PIN-based OAuth authentication and server discovery,
// see the [github.com/0skillallluck/scanline/provider/plex/auth] package.
package plex
