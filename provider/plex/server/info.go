package server

import "context"

// Info retrieves detailed information about the Plex Media Server.
//
// This includes the server name, version, platform, and configuration details.
func (s *Server) Info(ctx context.Context) (*ServerInfo, error) {
	var resp serverInfoContainer
	err := s.Get(ctx, "/").
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return &resp.MediaContainer, nil
}
