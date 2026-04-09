package player

import "testing"

func TestSettingsStateTranscodeParamsAllowsRawDirectPlayWithoutSubtitleChoices(t *testing.T) {
	t.Parallel()

	state := &settingsState{
		audioStreamIDs:    []int{11},
		subtitleStreamIDs: []int{0},
	}

	if got := state.transcodeParams(0, 0, 0); got != nil {
		t.Fatalf("transcodeParams() = %#v, want nil", got)
	}
}

func TestSettingsStateTranscodeParamsKeepsManagedPlaybackWhenSubtitleChoicesExist(t *testing.T) {
	t.Parallel()

	state := &settingsState{
		audioStreamIDs:    []int{11},
		subtitleStreamIDs: []int{0, 27},
	}

	got := state.transcodeParams(0, 0, 0)
	if got == nil {
		t.Fatal("transcodeParams() = nil, want non-nil")
	}
	if got.SubtitleStreamID != 0 {
		t.Fatalf("SubtitleStreamID = %d, want 0", got.SubtitleStreamID)
	}
	if !got.DirectStreamAudio {
		t.Fatal("DirectStreamAudio = false, want true")
	}
}

func TestSettingsStateTranscodeParamsIncludesSelectedSubtitleTrack(t *testing.T) {
	t.Parallel()

	state := &settingsState{
		audioStreamIDs:    []int{11},
		subtitleStreamIDs: []int{0, 27},
	}

	got := state.transcodeParams(0, 0, 1)
	if got == nil {
		t.Fatal("transcodeParams() = nil, want non-nil")
	}
	if got.SubtitleStreamID != 27 {
		t.Fatalf("SubtitleStreamID = %d, want 27", got.SubtitleStreamID)
	}
}
