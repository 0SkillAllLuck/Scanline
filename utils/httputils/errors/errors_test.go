package errors

import (
	"errors"
	"testing"
)

func TestHTTPError_ErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      *HTTPError
		contains string
	}{
		{
			name: "client error",
			err: &HTTPError{
				StatusCode: 400,
				Status:     "400 Bad Request",
				Body:       []byte("invalid request"),
				Kind:       ErrKindClient,
			},
			contains: "400 Bad Request",
		},
		{
			name: "server error",
			err: &HTTPError{
				StatusCode: 500,
				Status:     "500 Internal Server Error",
				Body:       []byte("server error"),
				Kind:       ErrKindServer,
			},
			contains: "500 Internal Server Error",
		},
		{
			name: "authentication error",
			err: &HTTPError{
				StatusCode: 401,
				Status:     "401 Unauthorized",
				Body:       []byte("unauthorized"),
				Kind:       ErrKindAuthentication,
			},
			contains: "401 Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if msg == "" {
				t.Error("Error() returned empty string")
			}
			if !contains(msg, tt.contains) {
				t.Errorf("Error() = %q, want to contain %q", msg, tt.contains)
			}
		})
	}
}

func TestHTTPError_Is_MatchesKind(t *testing.T) {
	tests := []struct {
		name   string
		err    *HTTPError
		target error
		want   bool
	}{
		{
			name: "401 matches ErrAuthentication",
			err: &HTTPError{
				StatusCode: 401,
				Kind:       ErrKindAuthentication,
			},
			target: ErrAuthentication,
			want:   true,
		},
		{
			name: "403 matches ErrAuthorization",
			err: &HTTPError{
				StatusCode: 403,
				Kind:       ErrKindAuthorization,
			},
			target: ErrAuthorization,
			want:   true,
		},
		{
			name: "404 matches ErrNotFound",
			err: &HTTPError{
				StatusCode: 404,
				Kind:       ErrKindNotFound,
			},
			target: ErrNotFound,
			want:   true,
		},
		{
			name: "500 matches ErrServer",
			err: &HTTPError{
				StatusCode: 500,
				Kind:       ErrKindServer,
			},
			target: ErrServer,
			want:   true,
		},
		{
			name: "401 does not match ErrNotFound",
			err: &HTTPError{
				StatusCode: 401,
				Kind:       ErrKindAuthentication,
			},
			target: ErrNotFound,
			want:   false,
		},
		{
			name: "400 matches ErrClient",
			err: &HTTPError{
				StatusCode: 400,
				Kind:       ErrKindClient,
			},
			target: ErrClient,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	httpErr := &HTTPError{
		StatusCode: 500,
		Status:     "500 Internal Server Error",
		Kind:       ErrKindServer,
		Wrapped:    baseErr,
	}

	if !errors.Is(httpErr, baseErr) {
		t.Error("HTTPError should unwrap to base error")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
