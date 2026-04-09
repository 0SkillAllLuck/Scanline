package player

import "testing"

func TestResumeOffsetSeconds(t *testing.T) {
	t.Parallel()

	if got := resumeOffsetSeconds(0); got != 0 {
		t.Fatalf("resumeOffsetSeconds(0) = %d, want 0", got)
	}
	if got := resumeOffsetSeconds(999999); got != 0 {
		t.Fatalf("resumeOffsetSeconds(999999) = %d, want 0", got)
	}
	if got := resumeOffsetSeconds(12345678); got != 12 {
		t.Fatalf("resumeOffsetSeconds(12345678) = %d, want 12", got)
	}
}

func TestPlaybackTimestampUs(t *testing.T) {
	t.Parallel()

	if got := playbackTimestampUs(42_000_000, 3_000_000); got != 45_000_000 {
		t.Fatalf("playbackTimestampUs() = %d, want %d", got, 45_000_000)
	}
	if got := playbackTimestampUs(42_000_000, -1); got != 42_000_000 {
		t.Fatalf("playbackTimestampUs() with negative media ts = %d, want %d", got, 42_000_000)
	}
}

func TestPlaybackDurationUs(t *testing.T) {
	t.Parallel()

	if got := playbackDurationUs(3_600_000_000, 1_800_000_000, 0); got != 3_600_000_000 {
		t.Fatalf("playbackDurationUs() with unknown media duration = %d, want %d", got, 3_600_000_000)
	}
	if got := playbackDurationUs(3_600_000_000, 1_800_000_000, 1_800_000_000); got != 3_600_000_000 {
		t.Fatalf("playbackDurationUs() = %d, want %d", got, 3_600_000_000)
	}
	if got := playbackDurationUs(0, 1_800_000_000, 1_800_000_000); got != 3_600_000_000 {
		t.Fatalf("playbackDurationUs() without known duration = %d, want %d", got, 3_600_000_000)
	}
}

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
	if !got.SubtitleSelectionExplicit {
		t.Fatal("SubtitleSelectionExplicit = false, want true")
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
	if !got.SubtitleSelectionExplicit {
		t.Fatal("SubtitleSelectionExplicit = false, want true")
	}
}
