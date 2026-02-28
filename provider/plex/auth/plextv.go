package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Resource represents a Plex resource (server, player, etc.) discovered from plex.tv.
type Resource struct {
	// Name is the friendly name of the resource.
	Name string `json:"name"`

	// Product is the product name (e.g., "Plex Media Server").
	Product string `json:"product"`

	// ProductVersion is the version of the product.
	ProductVersion string `json:"productVersion"`

	// Platform is the operating system platform.
	Platform string `json:"platform"`

	// PlatformVersion is the version of the platform.
	PlatformVersion string `json:"platformVersion"`

	// Device is the device type.
	Device string `json:"device"`

	// ClientIdentifier is the unique identifier for this resource.
	ClientIdentifier string `json:"clientIdentifier"`

	// Provides indicates what services this resource provides.
	Provides string `json:"provides"`

	// Owned indicates if the current user owns this resource.
	Owned bool `json:"owned"`

	// PublicAddress is the public IP address of the resource.
	PublicAddress string `json:"publicAddress,omitempty"`

	// AccessToken is the access token for this resource.
	AccessToken string `json:"accessToken,omitempty"`

	// SourceTitle is the title of the source (for shared resources).
	SourceTitle string `json:"sourceTitle,omitempty"`

	// HTTPSRequired indicates if HTTPS is required for connections.
	HTTPSRequired bool `json:"httpsRequired"`

	// Relay indicates if the resource is accessible via relay.
	Relay bool `json:"relay"`

	// PublicAddressMatches indicates if the client's public address matches.
	PublicAddressMatches bool `json:"publicAddressMatches"`

	// Presence indicates if the resource is currently online.
	Presence bool `json:"presence"`

	// Connections contains the available connection endpoints.
	Connections []ResourceConnection `json:"connections"`
}

// ResourceConnection represents a connection endpoint for a Plex resource.
type ResourceConnection struct {
	// Protocol is the connection protocol (http or https).
	Protocol string `json:"protocol"`

	// Address is the IP address or hostname.
	Address string `json:"address"`

	// Port is the connection port.
	Port int `json:"port"`

	// URI is the full connection URI.
	URI string `json:"uri"`

	// Local indicates if this is a local network connection.
	Local bool `json:"local"`

	// Relay indicates if this connection goes through a relay.
	Relay bool `json:"relay"`

	// IPv6 indicates if this is an IPv6 address.
	IPv6 bool `json:"IPv6"`
}

// PlexTV provides access to plex.tv API endpoints for server discovery.
type PlexTV struct {
	clientIdentifier string
	httpClient       *http.Client
}

// NewPlexTV creates a new PlexTV client with the given client identifier.
func NewPlexTV(clientIdentifier string) *PlexTV {
	return &PlexTV{
		clientIdentifier: clientIdentifier,
		httpClient:       http.DefaultClient,
	}
}

// DiscoverServers retrieves the list of available Plex Media Servers for the authenticated user.
// The token parameter is the authentication token obtained from PIN authentication.
func (p *PlexTV) DiscoverServers(ctx context.Context, token string) ([]Resource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, plexTVBaseURL+"/api/v2/resources", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("X-Plex-Client-Identifier", p.clientIdentifier)

	params := req.URL.Query()
	params.Set("includeHttps", "1")
	params.Set("includeRelay", "1")
	req.URL.RawQuery = params.Encode()

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to discover servers: %s", resp.Status)
	}

	var resources []Resource
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return resources, nil
}

// DiscoverServers is a convenience function that discovers servers using the default client.
func DiscoverServers(ctx context.Context, token, clientIdentifier string) ([]Resource, error) {
	return NewPlexTV(clientIdentifier).DiscoverServers(ctx, token)
}

// User represents a Plex account user.
type User struct {
	Username string `json:"username"`
	Title    string `json:"title"`
	Email    string `json:"email"`
}

// GetUser retrieves the Plex account information for the authenticated user.
func (p *PlexTV) GetUser(ctx context.Context, token string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, plexTVBaseURL+"/api/v2/user", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("X-Plex-Client-Identifier", p.clientIdentifier)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user: %s", resp.Status)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &user, nil
}

// GetUser is a convenience function that retrieves the user using the default client.
func GetUser(ctx context.Context, token, clientIdentifier string) (*User, error) {
	return NewPlexTV(clientIdentifier).GetUser(ctx, token)
}
