package player

import (
	"testing"

	"github.com/0skillallluck/scanline/app/sources"
	"github.com/go-gst/go-gst/gst"
)

func TestRedactToken(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no token",
			in:   "https://plex.local/library/parts/42?session=abc",
			want: "https://plex.local/library/parts/42?session=abc",
		},
		{
			name: "single token in query string",
			in:   "https://plex.local/library?X-Plex-Token=secret123",
			want: "https://plex.local/library?X-Plex-Token=REDACTED",
		},
		{
			name: "token followed by another param",
			in:   "https://plex.local/library?X-Plex-Token=secret123&session=abc",
			want: "https://plex.local/library?X-Plex-Token=REDACTED&session=abc",
		},
		{
			name: "token in middle of query string",
			in:   "https://plex.local/library?session=abc&X-Plex-Token=secret123&offset=10",
			want: "https://plex.local/library?session=abc&X-Plex-Token=REDACTED&offset=10",
		},
		{
			name: "token quoted in error message",
			in:   `Could not open resource "https://plex.local/?X-Plex-Token=secret123" for reading`,
			want: `Could not open resource "https://plex.local/?X-Plex-Token=REDACTED" for reading`,
		},
		{
			name: "token followed by space in log line",
			in:   "url=https://plex.local/?X-Plex-Token=secret123 transcoding=false",
			want: "url=https://plex.local/?X-Plex-Token=REDACTED transcoding=false",
		},
		{
			name: "multiple tokens in same string",
			in:   "first=https://a/?X-Plex-Token=aaa&q=1 second=https://b/?X-Plex-Token=bbb",
			want: "first=https://a/?X-Plex-Token=REDACTED&q=1 second=https://b/?X-Plex-Token=REDACTED",
		},
		{
			name: "empty token value",
			in:   "https://plex.local/library?X-Plex-Token=&other=1",
			want: "https://plex.local/library?X-Plex-Token=REDACTED&other=1",
		},
		{
			name: "token followed by URL fragment",
			in:   "https://plex.local/library?X-Plex-Token=secret123#section",
			want: "https://plex.local/library?X-Plex-Token=REDACTED#section",
		},
		{
			name: "token in single-quoted error message",
			in:   "open 'https://plex.local/?X-Plex-Token=secret123' failed",
			want: "open 'https://plex.local/?X-Plex-Token=REDACTED' failed",
		},
		{
			name: "token followed by newline in multiline log",
			in:   "uri=https://plex.local/?X-Plex-Token=secret123\nstatus=ok",
			want: "uri=https://plex.local/?X-Plex-Token=REDACTED\nstatus=ok",
		},
		{
			name: "token followed by tab in tab-separated log",
			in:   "uri=https://plex.local/?X-Plex-Token=secret123\tstatus=ok",
			want: "uri=https://plex.local/?X-Plex-Token=REDACTED\tstatus=ok",
		},
		{
			name: "token in angle-bracket log fragment",
			in:   "<url=https://plex.local/?X-Plex-Token=secret123>",
			want: "<url=https://plex.local/?X-Plex-Token=REDACTED>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactToken(tt.in)
			if got != tt.want {
				t.Errorf("redactToken(%q):\n  got  %q\n  want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestInitialManagedPlaybackParamsExternalSubtitle(t *testing.T) {
	params := PlayerParams{
		RatingKey:  "99",
		ViewOffset: 42_000,
		Media: []sources.Media{
			{
				Part: []sources.Part{
					{
						Stream: []sources.Stream{
							{ID: 10, StreamType: sources.StreamTypeAudio, Selected: true},
							{ID: 20, StreamType: sources.StreamTypeSubtitle, Selected: true, Key: "/library/streams/20"},
						},
					},
				},
			},
		},
	}

	got := initialManagedPlaybackParams(params, "session-id")
	if got == nil {
		t.Fatal("initialManagedPlaybackParams returned nil for selected external subtitle")
	}
	if got.RatingKey != "99" || got.SessionID != "session-id" {
		t.Fatalf("unexpected identity fields: %#v", got)
	}
	if !got.DirectStreamAudio || got.AudioStreamID != 10 {
		t.Fatalf("unexpected audio settings: %#v", got)
	}
	if got.SubtitleStreamID != 20 || !got.SubtitleSelectionExplicit {
		t.Fatalf("unexpected subtitle settings: %#v", got)
	}
	if got.Offset != 42 {
		t.Fatalf("Offset = %d, want 42", got.Offset)
	}
}

func TestInitialManagedPlaybackParamsEmbeddedSubtitleDirect(t *testing.T) {
	params := PlayerParams{
		RatingKey: "99",
		Media: []sources.Media{
			{
				Part: []sources.Part{
					{
						Stream: []sources.Stream{
							{ID: 10, StreamType: sources.StreamTypeAudio, Selected: true},
							{ID: 20, StreamType: sources.StreamTypeSubtitle, Selected: true},
						},
					},
				},
			},
		},
	}

	if got := initialManagedPlaybackParams(params, "session-id"); got != nil {
		t.Fatalf("initialManagedPlaybackParams returned %#v for embedded subtitle", got)
	}
}

func TestTranscodeParamsExternalSubtitleNeedsManagedPlayback(t *testing.T) {
	state := settingsState{
		params:                        PlayerParams{RatingKey: "99"},
		sessionID:                     "session-id",
		audioStreamIDs:                []int{10},
		subtitleStreamIDs:             []int{0, 20},
		subtitleManagedPlaybackNeeded: []bool{false, true},
	}

	got := state.transcodeParams(0, 0, 1)
	if got == nil {
		t.Fatal("transcodeParams returned nil for external subtitle")
	}
	if got.AudioStreamID != 10 || got.SubtitleStreamID != 20 {
		t.Fatalf("unexpected stream IDs: %#v", got)
	}
	if !got.DirectStreamAudio || !got.SubtitleSelectionExplicit {
		t.Fatalf("unexpected transcode flags: %#v", got)
	}
}

func TestTranscodeParamsEmbeddedSubtitleStaysDirect(t *testing.T) {
	state := settingsState{
		params:                        PlayerParams{RatingKey: "99"},
		sessionID:                     "session-id",
		audioStreamIDs:                []int{10},
		subtitleStreamIDs:             []int{0, 20},
		subtitleManagedPlaybackNeeded: []bool{false, false},
	}

	if got := state.transcodeParams(0, 0, 1); got != nil {
		t.Fatalf("transcodeParams returned %#v for embedded subtitle", got)
	}
}

func TestTranscodeParamsMissingAudioSelectionStaysDirect(t *testing.T) {
	state := settingsState{
		params:                        PlayerParams{RatingKey: "99"},
		sessionID:                     "session-id",
		subtitleStreamIDs:             []int{0},
		subtitleManagedPlaybackNeeded: []bool{false},
	}

	if got := state.transcodeParams(0, -1, 0); got != nil {
		t.Fatalf("transcodeParams returned %#v for missing audio selection", got)
	}
}

func TestRefreshSelectedFlagsUsesStreamsSelectedList(t *testing.T) {
	ensureGstInit()

	audioCaps := gst.NewCapsFromString("audio/x-raw")
	if audioCaps == nil {
		t.Skip("could not create audio caps")
	}
	defer audioCaps.Unref()

	videoCaps := gst.NewCapsFromString("video/x-raw")
	if videoCaps == nil {
		t.Skip("could not create video caps")
	}
	defer videoCaps.Unref()

	videoStream := gst.NewStream("video", videoCaps, gst.StreamTypeVideo, gst.StreamFlagSelect)
	defaultAudio := gst.NewStream("audio-default", audioCaps, gst.StreamTypeAudio, gst.StreamFlagSelect)
	selectedAudio := gst.NewStream("audio-selected", audioCaps, gst.StreamTypeAudio, 0)
	collection := gst.NewStreamCollection("test")
	if err := collection.AddStream(videoStream); err != nil {
		t.Fatal(err)
	}
	if err := collection.AddStream(defaultAudio); err != nil {
		t.Fatal(err)
	}
	if err := collection.AddStream(selectedAudio); err != nil {
		t.Fatal(err)
	}

	src, err := gst.NewElement("fakesrc")
	if err != nil {
		t.Skipf("fakesrc unavailable: %v", err)
	}
	msg := gst.NewStreamSelectedMessage(src, collection)
	if msg == nil {
		t.Fatal("NewStreamSelectedMessage returned nil")
	}
	msg.StreamsSelectedAdd(videoStream)
	msg.StreamsSelectedAdd(selectedAudio)

	c := &core{
		videoStreams: []streamInfo{{StreamID: "video", Selected: true}},
		audioStreams: []streamInfo{
			{StreamID: "audio-default", Selected: true},
			{StreamID: "audio-selected", Selected: false},
		},
	}

	c.refreshSelectedFlags(msg)

	if c.audioStreams[0].Selected {
		t.Fatal("default audio stream stayed selected")
	}
	if !c.audioStreams[1].Selected {
		t.Fatal("message-selected audio stream was not marked selected")
	}
	if !c.videoStreams[0].Selected {
		t.Fatal("message-selected video stream was not marked selected")
	}
}
