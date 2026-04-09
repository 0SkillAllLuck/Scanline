package plex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildTranscodeQueryExplicitSubtitleOff(t *testing.T) {
	t.Parallel()

	client := NewClient("http://localhost:32400", "token", "client-id")
	query := client.BuildTranscodeQuery(TranscodeParams{
		RatingKey:                 "42",
		SessionID:                 "session",
		DirectStreamAudio:         true,
		SubtitleStreamID:          0,
		SubtitleSelectionExplicit: true,
	})

	if got := query.Get("subtitleStreamID"); got != "0" {
		t.Fatalf("subtitleStreamID = %q, want %q", got, "0")
	}
	if got := query.Get("subtitles"); got != "none" {
		t.Fatalf("subtitles = %q, want %q", got, "none")
	}
}

func TestClientSelectStreamsAllowsSubtitleOff(t *testing.T) {
	t.Parallel()

	var gotMethod string
	var gotPath string
	var gotAudio string
	var gotSubtitle string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAudio = r.URL.Query().Get("audioStreamID")
		gotSubtitle = r.URL.Query().Get("subtitleStreamID")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", "client-id")

	audioID := 17
	subtitleID := 0
	if err := client.SelectStreams(context.Background(), 123, &audioID, &subtitleID); err != nil {
		t.Fatalf("SelectStreams() error = %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodPut)
	}
	if gotPath != "/library/parts/123" {
		t.Fatalf("path = %q, want %q", gotPath, "/library/parts/123")
	}
	if gotAudio != "17" {
		t.Fatalf("audioStreamID = %q, want %q", gotAudio, "17")
	}
	if gotSubtitle != "0" {
		t.Fatalf("subtitleStreamID = %q, want %q", gotSubtitle, "0")
	}
}
