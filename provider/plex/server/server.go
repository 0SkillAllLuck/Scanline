// Package server provides access to Plex server information endpoints.
package server

import "github.com/0skillallluck/scanline/provider/plex/base"

// Server provides access to server information endpoints.
type Server struct {
	*base.Base
}

// New creates a new Server service.
func New(b *base.Base) *Server {
	return &Server{Base: b}
}

// ServerInfo contains detailed information about a Plex Media Server.
type ServerInfo struct {
	// Size is the number of items in the response.
	Size int `json:"size"`

	// MachineIdentifier is the unique identifier for this server.
	MachineIdentifier string `json:"machineIdentifier"`

	// FriendlyName is the user-configured name of the server.
	FriendlyName string `json:"friendlyName"`

	// Version is the Plex Media Server version.
	Version string `json:"version"`

	// Platform is the operating system (e.g., "Linux", "Windows").
	Platform string `json:"platform"`

	// PlatformVersion is the OS version.
	PlatformVersion string `json:"platformVersion"`

	// Multiuser indicates if the server supports multiple users.
	Multiuser bool `json:"multiuser"`

	// MyPlex indicates if the server is connected to plex.tv.
	MyPlex bool `json:"myPlex"`

	// MyPlexUsername is the plex.tv username of the server owner.
	MyPlexUsername string `json:"myPlexUsername"`

	// TranscoderActiveVideoSessions is the number of active transcode sessions.
	TranscoderActiveVideoSessions int `json:"transcoderActiveVideoSessions"`
}

// ServerIdentity contains minimal identification information for a server.
type ServerIdentity struct {
	// MachineIdentifier is the unique identifier for this server.
	MachineIdentifier string `json:"machineIdentifier"`

	// Version is the Plex Media Server version.
	Version string `json:"version"`
}

// Container types for JSON unmarshaling.

type serverInfoContainer struct {
	MediaContainer ServerInfo `json:"MediaContainer"`
}

type serverIdentityContainer struct {
	MediaContainer ServerIdentity `json:"MediaContainer"`
}
