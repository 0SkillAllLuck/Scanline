package auth

import "net/http"

// Strategy defines the interface for authentication methods.
// Implementations should modify the request to include authentication credentials.
type Strategy interface {
	// Authenticate modifies the request to include authentication credentials.
	// Returns an error if authentication cannot be applied.
	Authenticate(req *http.Request) error
}
