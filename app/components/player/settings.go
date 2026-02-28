package player

import (
	"context"
	"fmt"
	"log/slog"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

type qualityPreset struct {
	Label         string
	MaxBitrate    int    // kbps, 0 = original
	MaxResolution string // "WxH", empty = original
	DirectPlay    bool
}

var qualityPresets = []qualityPreset{
	{Label: "Original", DirectPlay: true},
	{Label: "20 Mbps 1080p", MaxBitrate: 20000, MaxResolution: "1920x1080"},
	{Label: "12 Mbps 1080p", MaxBitrate: 12000, MaxResolution: "1920x1080"},
	{Label: "10 Mbps 720p", MaxBitrate: 10000, MaxResolution: "1280x720"},
	{Label: "4 Mbps 720p", MaxBitrate: 4000, MaxResolution: "1280x720"},
	{Label: "2 Mbps 480p", MaxBitrate: 2000, MaxResolution: "854x480"},
	{Label: "0.7 Mbps 320p", MaxBitrate: 700, MaxResolution: "480x320"},
}

type settingsState struct {
	params            PlayerParams
	source            sources.Source
	sessionID         string
	audioStreamIDs    []int // dropdown index → Stream.ID
	subtitleStreamIDs []int // index 0 = "None" (ID 0), rest from metadata
}

// transcodeParams builds TranscodeParams from UI selections.
// Returns nil if direct play should be used instead.
func (s *settingsState) transcodeParams(qualityIdx, audioIdx, subtitleIdx int) *sources.TranscodeParams {
	preset := qualityPresets[qualityIdx]
	audioID := 0
	if audioIdx >= 0 && audioIdx < len(s.audioStreamIDs) {
		audioID = s.audioStreamIDs[audioIdx]
	}
	subtitleID := 0
	if subtitleIdx >= 0 && subtitleIdx < len(s.subtitleStreamIDs) {
		subtitleID = s.subtitleStreamIDs[subtitleIdx]
	}

	// Direct play if original quality, first audio track, and no subtitles
	if preset.DirectPlay && audioIdx == 0 && subtitleID == 0 {
		return nil
	}

	return &sources.TranscodeParams{
		RatingKey:         s.params.RatingKey,
		SessionID:         s.sessionID,
		DirectStreamAudio: preset.DirectPlay,
		MaxBitrate:        preset.MaxBitrate,
		MaxResolution:     preset.MaxResolution,
		AudioStreamID:     audioID,
		SubtitleStreamID:  subtitleID,
	}
}

func streamLabel(stream sources.Stream, index int) string {
	if stream.DisplayTitle != "" {
		return stream.DisplayTitle
	}
	if stream.Language != "" {
		return stream.Language
	}
	return fmt.Sprintf("Track %d", index+1)
}

func buildSettingsPopover(
	params PlayerParams,
	src sources.Source,
	sessionID string,
	onChanged func(newURL string, transcodeParams *sources.TranscodeParams),
) *gtk.Popover {
	streams := params.Media[0].Part[0].Stream

	// Build quality labels
	qualityLabels := make([]string, len(qualityPresets))
	for i, p := range qualityPresets {
		qualityLabels[i] = p.Label
	}

	// Build audio stream labels and IDs
	var audioLabels []string
	var audioStreamIDs []int
	selectedAudio := uint(0)
	audioIdx := 0
	for i, s := range streams {
		if s.StreamType == 2 { // audio
			audioLabels = append(audioLabels, streamLabel(s, i))
			audioStreamIDs = append(audioStreamIDs, s.ID)
			if s.Selected {
				selectedAudio = uint(audioIdx)
			}
			audioIdx++
		}
	}

	// Build subtitle stream labels and IDs ("None" + subtitle streams)
	subtitleLabels := []string{gettext.Get("None")}
	subtitleStreamIDs := []int{0}
	selectedSubtitle := uint(0)
	subtitleIdx := 0
	for i, s := range streams {
		if s.StreamType == 3 { // subtitle
			subtitleLabels = append(subtitleLabels, streamLabel(s, i))
			subtitleStreamIDs = append(subtitleStreamIDs, s.ID)
			subtitleIdx++
			if s.Selected {
				selectedSubtitle = uint(subtitleIdx)
			}
		}
	}

	state := &settingsState{
		params:            params,
		source:            src,
		sessionID:         sessionID,
		audioStreamIDs:    audioStreamIDs,
		subtitleStreamIDs: subtitleStreamIDs,
	}

	qualityDD := gtk.NewDropDownFromStrings(qualityLabels)
	qualityDD.SetSelected(0) // Original

	audioDD := gtk.NewDropDownFromStrings(audioLabels)
	if len(audioLabels) > 0 {
		audioDD.SetSelected(selectedAudio)
	}

	subtitleDD := gtk.NewDropDownFromStrings(subtitleLabels)
	subtitleDD.SetSelected(selectedSubtitle)

	// Build content and create popover first so fireChange can reference it
	content := VStack(
		Label(gettext.Get("Quality")).WithCSSClass("heading").HAlign(gtk.AlignStartValue),
		Widget(&qualityDD.Widget),
		Label(gettext.Get("Audio")).WithCSSClass("heading").HAlign(gtk.AlignStartValue).MarginTop(12),
		Widget(&audioDD.Widget),
		Label(gettext.Get("Subtitles")).WithCSSClass("heading").HAlign(gtk.AlignStartValue).MarginTop(12),
		Widget(&subtitleDD.Widget),
	).Spacing(4).HMargin(4).VMargin(4)

	popover := Popover(content)
	rawPopover := popover()

	fireChange := func() {
		rawPopover.Popdown()

		qi := int(qualityDD.GetSelected())
		ai := int(audioDD.GetSelected())
		si := int(subtitleDD.GetSelected())
		slog.Debug("player settings changed",
			"quality", qualityPresets[qi].Label,
			"audioIdx", ai,
			"subtitleIdx", si,
		)

		params := state.transcodeParams(qi, ai, si)
		if params == nil {
			// Direct play
			onChanged(state.source.StreamURL(state.params.PartKey), nil)
			return
		}

		// Transcode: call decision endpoint first (in background), then switch stream
		q := state.source.BuildTranscodeQuery(*params)
		startURL := state.source.TranscodeStartURL(q)
		go func() {
			if err := state.source.MakeTranscodeDecision(context.Background(), q); err != nil {
				slog.Error("player: decision call failed", "error", err)
				return
			}
			schwifty.OnMainThreadOncePure(func() {
				onChanged(startURL, params)
			})
		}()
	}

	qualityDD.ConnectSignal("notify::selected", new(func() {
		fireChange()
	}))
	audioDD.ConnectSignal("notify::selected", new(func() {
		fireChange()
	}))
	subtitleDD.ConnectSignal("notify::selected", new(func() {
		fireChange()
	}))

	return rawPopover
}
