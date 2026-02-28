package request

import (
	"encoding/json"
	"net/http"

	"github.com/0skillallluck/scanline/utils/httputils/response"
)

type cachedResponse struct {
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

func marshalResponse(r *response.Response) ([]byte, error) {
	headers := make(map[string]string)
	for k, v := range r.Headers {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return json.Marshal(&cachedResponse{
		StatusCode: r.StatusCode,
		Status:     r.Status,
		Headers:    headers,
		Body:       r.Body,
	})
}

func unmarshalResponse(data []byte) (*response.Response, error) {
	var cached cachedResponse
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}

	headers := make(http.Header)
	for k, v := range cached.Headers {
		headers.Set(k, v)
	}

	return &response.Response{
		StatusCode: cached.StatusCode,
		Status:     cached.Status,
		Headers:    headers,
		Body:       cached.Body,
	}, nil
}
