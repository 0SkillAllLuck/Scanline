package response

import (
	"errors"
	"net/http"
	"testing"

	httperrors "github.com/0skillallluck/scanline/utils/httputils/errors"
)

func TestResponse_JSON_DecodesCorrectly(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name    string
		body    []byte
		want    testStruct
		wantErr bool
	}{
		{
			name: "valid JSON",
			body: []byte(`{"name":"test","value":42}`),
			want: testStruct{Name: "test", Value: 42},
		},
		{
			name:    "invalid JSON",
			body:    []byte(`{invalid`),
			wantErr: true,
		},
		{
			name: "empty JSON object",
			body: []byte(`{}`),
			want: testStruct{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{
				StatusCode: 200,
				Body:       tt.body,
			}

			var got testStruct
			err := resp.JSON(&got)

			if (err != nil) != tt.wantErr {
				t.Errorf("JSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("JSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResponse_String_ReturnsBody(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{
			name: "simple string",
			body: []byte("hello world"),
			want: "hello world",
		},
		{
			name: "empty body",
			body: []byte{},
			want: "",
		},
		{
			name: "JSON body",
			body: []byte(`{"key":"value"}`),
			want: `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{Body: tt.body}
			if got := resp.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResponse_Bytes_ReturnsBody(t *testing.T) {
	body := []byte("test body")
	resp := &Response{Body: body}

	got := resp.Bytes()
	if string(got) != string(body) {
		t.Errorf("Bytes() = %v, want %v", got, body)
	}
}

func TestResponse_IsSuccess_TrueFor2xx(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"200 OK", 200, true},
		{"201 Created", 201, true},
		{"204 No Content", 204, true},
		{"299 edge case", 299, true},
		{"199 not success", 199, false},
		{"300 redirect", 300, false},
		{"400 client error", 400, false},
		{"500 server error", 500, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{StatusCode: tt.statusCode}
			if got := resp.IsSuccess(); got != tt.want {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResponse_CheckStatus_ReturnsErrorFor4xx5xx(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		status     string
		wantErr    bool
		errKind    httperrors.ErrorKind
	}{
		{
			name:       "200 OK no error",
			statusCode: 200,
			status:     "200 OK",
			wantErr:    false,
		},
		{
			name:       "401 returns authentication error",
			statusCode: 401,
			status:     "401 Unauthorized",
			wantErr:    true,
			errKind:    httperrors.ErrKindAuthentication,
		},
		{
			name:       "403 returns authorization error",
			statusCode: 403,
			status:     "403 Forbidden",
			wantErr:    true,
			errKind:    httperrors.ErrKindAuthorization,
		},
		{
			name:       "404 returns not found error",
			statusCode: 404,
			status:     "404 Not Found",
			wantErr:    true,
			errKind:    httperrors.ErrKindNotFound,
		},
		{
			name:       "400 returns client error",
			statusCode: 400,
			status:     "400 Bad Request",
			wantErr:    true,
			errKind:    httperrors.ErrKindClient,
		},
		{
			name:       "500 returns server error",
			statusCode: 500,
			status:     "500 Internal Server Error",
			wantErr:    true,
			errKind:    httperrors.ErrKindServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{
				StatusCode: tt.statusCode,
				Status:     tt.status,
				Headers:    make(http.Header),
				Body:       []byte("test body"),
			}

			err := resp.CheckStatus()

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				var httpErr *httperrors.HTTPError
				if !errors.As(err, &httpErr) {
					t.Errorf("CheckStatus() error is not HTTPError")
					return
				}
				if httpErr.Kind != tt.errKind {
					t.Errorf("CheckStatus() error kind = %v, want %v", httpErr.Kind, tt.errKind)
				}
			}
		})
	}
}
