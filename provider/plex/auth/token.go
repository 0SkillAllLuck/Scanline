package auth

import "net/http"

// TokenStrategy implements token-based authentication for Plex APIs.
// It adds the X-Plex-Token header to all requests.
type TokenStrategy struct {
	token string
}

// NewTokenStrategy creates a new TokenStrategy with the given authentication token.
func NewTokenStrategy(token string) *TokenStrategy {
	return &TokenStrategy{
		token: token,
	}
}

// Authenticate adds the X-Plex-Token header to the request.
func (s *TokenStrategy) Authenticate(req *http.Request) error {
	req.Header.Set("X-Plex-Token", s.token)
	return nil
}
