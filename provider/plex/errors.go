package plex

import "fmt"

// APIError represents a generic error returned by the Plex API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("plex api error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("plex api error (status %d)", e.StatusCode)
}

// AuthenticationError indicates that the request was not authenticated.
// This typically means the token is missing, invalid, or expired.
type AuthenticationError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *AuthenticationError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("authentication failed (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("authentication failed (status %d)", e.StatusCode)
}

func (e *AuthenticationError) Is(target error) bool {
	_, ok := target.(*AuthenticationError)
	return ok
}

// AuthorizationError indicates that the request was authenticated but not authorized.
// This means the user does not have permission to access the requested resource.
type AuthorizationError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *AuthorizationError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("authorization denied (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("authorization denied (status %d)", e.StatusCode)
}

func (e *AuthorizationError) Is(target error) bool {
	_, ok := target.(*AuthorizationError)
	return ok
}

// NotFoundError indicates that the requested resource was not found.
type NotFoundError struct {
	StatusCode int
	Message    string
	Path       string
}

func (e *NotFoundError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("resource not found: %s", e.Path)
	}
	return "resource not found"
}

func (e *NotFoundError) Is(target error) bool {
	_, ok := target.(*NotFoundError)
	return ok
}

// ServerError indicates a server-side error (5xx status codes).
type ServerError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *ServerError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("server error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("server error (status %d)", e.StatusCode)
}

func (e *ServerError) Is(target error) bool {
	_, ok := target.(*ServerError)
	return ok
}

// EmptyResultError indicates that a query returned no results.
// This is used when an API call succeeds but returns an empty collection
// where at least one result was expected.
type EmptyResultError struct {
	Resource string
	ID       string
}

func (e *EmptyResultError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

func (e *EmptyResultError) Is(target error) bool {
	_, ok := target.(*EmptyResultError)
	return ok
}

// Sentinel errors for use with errors.Is().
var (
	// ErrNotFound is returned when a resource is not found (404).
	ErrNotFound = &NotFoundError{}

	// ErrAuthentication is returned when authentication fails (401).
	ErrAuthentication = &AuthenticationError{}

	// ErrAuthorization is returned when authorization is denied (403).
	ErrAuthorization = &AuthorizationError{}

	// ErrServer is returned for server-side errors (5xx).
	ErrServer = &ServerError{}

	// ErrEmptyResult is returned when a query returns no results.
	ErrEmptyResult = &EmptyResultError{}
)
