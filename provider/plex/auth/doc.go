// Package auth provides authentication strategies and server discovery for Plex.
//
// # Authentication Strategies
//
// The package defines the [Strategy] interface that all authentication methods
// must implement. Two strategies are provided:
//
//   - [TokenStrategy]: Simple token-based authentication using X-Plex-Token header
//   - PIN-based OAuth: For user authorization via plex.tv
//
// # PIN-Based Authentication
//
// For applications that need user authorization, use the PIN flow:
//
//  1. Call [RequestPin] to get a PIN code
//  2. Direct user to [AuthAppURL] with the PIN code
//  3. Use [PollPin] to wait for user authorization
//  4. Store the returned token for future use
//
// Example:
//
//	pin, err := auth.RequestPin(clientID)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Visit: %s\n", auth.AuthAppURL(clientID, pin.Code))
//	token, err := auth.PollPin(ctx, pin.ID, pin.Code, clientID, 2*time.Second)
//
// # Server Discovery
//
// Use [PlexTV.DiscoverServers] to find available Plex Media Servers:
//
//	plextv := auth.NewPlexTV(clientID)
//	servers, err := plextv.DiscoverServers(ctx, token)
package auth
