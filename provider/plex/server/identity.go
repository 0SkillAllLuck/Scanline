package server

import "context"

// Identity retrieves minimal identification information about the server.
//
// This is a lightweight endpoint that only returns the machine identifier and version.
func (s *Server) Identity(ctx context.Context) (*ServerIdentity, error) {
	var resp serverIdentityContainer
	err := s.Get(ctx, "/identity").
		DoAndDecode(&resp)
	if err != nil {
		return nil, err
	}
	return &resp.MediaContainer, nil
}
