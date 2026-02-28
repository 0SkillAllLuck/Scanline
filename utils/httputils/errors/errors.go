package errors

import "fmt"

// ErrorKind represents the category of HTTP error.
type ErrorKind int

const (
	// ErrKindClient represents 4xx client errors (except auth-related).
	ErrKindClient ErrorKind = iota
	// ErrKindServer represents 5xx server errors.
	ErrKindServer
	// ErrKindAuthentication represents 401 Unauthorized errors.
	ErrKindAuthentication
	// ErrKindAuthorization represents 403 Forbidden errors.
	ErrKindAuthorization
	// ErrKindNotFound represents 404 Not Found errors.
	ErrKindNotFound
)

// Sentinel errors for use with errors.Is().
var (
	ErrAuthentication = &HTTPError{Kind: ErrKindAuthentication}
	ErrAuthorization  = &HTTPError{Kind: ErrKindAuthorization}
	ErrNotFound       = &HTTPError{Kind: ErrKindNotFound}
	ErrServer         = &HTTPError{Kind: ErrKindServer}
	ErrClient         = &HTTPError{Kind: ErrKindClient}
)

// HTTPError represents an HTTP error response.
type HTTPError struct {
	StatusCode int
	Status     string
	Body       []byte
	Kind       ErrorKind
	Wrapped    error
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	if e.Status != "" {
		return fmt.Sprintf("HTTP error: %s", e.Status)
	}
	return fmt.Sprintf("HTTP error: status code %d", e.StatusCode)
}

// Is implements errors.Is support for comparing error kinds.
func (e *HTTPError) Is(target error) bool {
	if t, ok := target.(*HTTPError); ok {
		return e.Kind == t.Kind
	}
	return false
}

// Unwrap returns the wrapped error, if any.
func (e *HTTPError) Unwrap() error {
	return e.Wrapped
}

// NewHTTPError creates an HTTPError from a status code.
func NewHTTPError(statusCode int, status string, body []byte) *HTTPError {
	kind := classifyStatusCode(statusCode)
	return &HTTPError{
		StatusCode: statusCode,
		Status:     status,
		Body:       body,
		Kind:       kind,
	}
}

// classifyStatusCode determines the ErrorKind for a given status code.
func classifyStatusCode(statusCode int) ErrorKind {
	switch statusCode {
	case 401:
		return ErrKindAuthentication
	case 403:
		return ErrKindAuthorization
	case 404:
		return ErrKindNotFound
	default:
		if statusCode >= 500 {
			return ErrKindServer
		}
		return ErrKindClient
	}
}
