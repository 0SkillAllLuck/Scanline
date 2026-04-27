package player

import (
	"context"
	"fmt"
	"log/slog"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
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
	params                        PlayerParams
	ctx                           context.Context
	source                        sources.Source
	sessionID                     string
	partID                        int
	audioStreamIDs                []int  // dropdown index → Stream.ID
	subtitleStreamIDs             []int  // index 0 = "None" (ID 0), rest from metadata
	subtitleManagedPlaybackNeeded []bool // aligned with subtitleStreamIDs
}

type initialStreamSelection struct {
	AudioIdx    int
	SubtitleIdx int
}

// transcodeParams builds TranscodeParams from UI selections.
// Returns nil if direct play should be used instead.
//
// Triggers for transcoding: a non-original video quality preset, a
// non-default audio track, or a server-managed subtitle track that direct
// play cannot see (external/sidecar subtitles). When a transcode IS happening,
// the chosen subtitle has to be carried into the session — Plex burns or muxes
// subtitles server-side based on SubtitleStreamID.
func (s *settingsState) transcodeParams(qualityIdx, audioIdx, subtitleIdx int) *sources.TranscodeParams {
	qualityIdx = normalizeQualityIndex(qualityIdx)
	preset := qualityPresets[qualityIdx]
	audioID := 0
	if audioIdx >= 0 && audioIdx < len(s.audioStreamIDs) {
		audioID = s.audioStreamIDs[audioIdx]
	}
	subtitleID := 0
	if subtitleIdx >= 0 && subtitleIdx < len(s.subtitleStreamIDs) {
		subtitleID = s.subtitleStreamIDs[subtitleIdx]
	}

	if preset.DirectPlay && audioIdx <= 0 && !s.subtitleRequiresManagedPlayback(subtitleIdx) {
		return nil
	}

	return &sources.TranscodeParams{
		RatingKey:                 s.params.RatingKey,
		SessionID:                 s.sessionID,
		DirectStreamAudio:         preset.DirectPlay,
		MaxBitrate:                preset.MaxBitrate,
		MaxResolution:             preset.MaxResolution,
		AudioStreamID:             audioID,
		SubtitleStreamID:          subtitleID,
		SubtitleSelectionExplicit: len(s.subtitleStreamIDs) > 1,
	}
}

func (s *settingsState) subtitleRequiresManagedPlayback(subtitleIdx int) bool {
	if subtitleIdx < 0 || subtitleIdx >= len(s.subtitleManagedPlaybackNeeded) {
		return false
	}
	return s.subtitleManagedPlaybackNeeded[subtitleIdx]
}

func streamRequiresManagedPlayback(stream sources.Stream) bool {
	return stream.StreamType == sources.StreamTypeSubtitle && (stream.External || stream.Key != "")
}

func normalizeQualityIndex(idx int) int {
	if idx < 0 || idx >= len(qualityPresets) {
		return 0
	}
	return idx
}

func selectedDropDownIndex(dd *gtk.DropDown) int {
	selected := dd.GetSelected()
	if selected == gtk.INVALID_LIST_POSITION {
		return -1
	}
	return int(selected)
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

func resumeOffsetSeconds(timestampMicroseconds int64) int {
	if timestampMicroseconds <= 0 {
		return 0
	}
	return int(timestampMicroseconds / 1000000)
}

// buildSettingsPopover constructs the in-player settings popover (quality,
// audio, subtitles).
//
// Returns the popover and the dropdown's initial stream indices so the caller
// can apply them to the player core once that's available — SetSelected on the
// dropdown is a no-op when the value doesn't actually change, so the
// notify::selected signal does NOT fire on first paint, and we'd otherwise drift
// out of sync with whatever default streams playbin3 picks.
//
// Index convention: -1 == "None" (subtitle rendering off); 0+ is the Nth
// text stream.
//
// Index ordering assumption: Plex returns container streams in declaration
// order, and playbin3's MessageStreamCollection enumerates them in the same
// order. So the dropdown's "first subtitle" lines up with playbin3 stream
// index 0. If Plex ever rearranges (e.g. to put the user-selected stream
// first), this mapping breaks and we'd need to match by language tag.
func buildSettingsPopover(
	params PlayerParams,
	src sources.Source,
	sessionID string,
	ctx context.Context,
	currentPosition func() int64,
	transcodeActive func() bool,
	onChanged func(newURL string, transcodeParams *sources.TranscodeParams),
	onAudioChanged func(audioStreamIdx int),
	onSubtitleChanged func(textStreamIdx int),
) (*gtk.Popover, initialStreamSelection) {
	streams := params.Media[0].Part[0].Stream

	// Build quality labels
	qualityLabels := make([]string, len(qualityPresets))
	for i, p := range qualityPresets {
		qualityLabels[i] = p.Label
	}

	// Audio dropdown: indices 0+ are audio streams in declaration order.
	// selectedAudio stays at 0 if no stream is marked Selected, which falls
	// back to the first audio stream — Plex's universal default.
	var audioLabels []string
	var audioStreamIDs []int
	var selectedAudio uint32
	for i, s := range streams {
		if s.StreamType != sources.StreamTypeAudio {
			continue
		}
		audioStreamIDs = append(audioStreamIDs, s.ID)
		audioLabels = append(audioLabels, streamLabel(s, i))
		if s.Selected {
			selectedAudio = uint32(len(audioLabels) - 1)
		}
	}

	// Subtitle dropdown: index 0 is "None" (sentinel ID 0), indices 1+ are
	// subtitle streams in declaration order. selectedSubtitle stays at 0 if
	// no stream is marked Selected, leaving "None" as the safe fallback.
	subtitleLabels := []string{gettext.Get("None")}
	subtitleStreamIDs := []int{0}
	subtitleManagedPlaybackNeeded := []bool{false}
	var selectedSubtitle uint32
	for i, s := range streams {
		if s.StreamType != sources.StreamTypeSubtitle {
			continue
		}
		subtitleStreamIDs = append(subtitleStreamIDs, s.ID)
		subtitleManagedPlaybackNeeded = append(subtitleManagedPlaybackNeeded, streamRequiresManagedPlayback(s))
		subtitleLabels = append(subtitleLabels, streamLabel(s, i))
		if s.Selected {
			selectedSubtitle = uint32(len(subtitleLabels) - 1)
		}
	}

	state := &settingsState{
		params:                        params,
		ctx:                           ctx,
		source:                        src,
		sessionID:                     sessionID,
		partID:                        params.Media[0].Part[0].ID,
		audioStreamIDs:                audioStreamIDs,
		subtitleStreamIDs:             subtitleStreamIDs,
		subtitleManagedPlaybackNeeded: subtitleManagedPlaybackNeeded,
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
	// GtkDropDown opens its own internal popover for the picker. When that
	// inner popover closes — whether by selection or by escape/click-outside
	// — GTK4 doesn't reliably hand the autohide grab back to this outer
	// popover, leaving it dismissible only via ESC. cascade-popdown closes
	// this popover whenever a child popover is dismissed, which sidesteps
	// the broken grab handling and matches the "I'm done picking" mental
	// model the user expects.
	rawPopover.SetCascadePopdown(true)

	// fireStreamSwitch handles quality/audio changes. These genuinely require
	// reopening the stream (a transcode session change or a direct-play swap
	// to pick up the new audio track), so we close the popover and rebuild
	// the URL.
	fireStreamSwitch := func() {
		rawPopover.Popdown()

		qi := normalizeQualityIndex(selectedDropDownIndex(qualityDD))
		ai := selectedDropDownIndex(audioDD)
		si := selectedDropDownIndex(subtitleDD)
		if si < 0 {
			si = 0
		}
		slog.Debug("player settings changed",
			"quality", qualityPresets[qi].Label,
			"audioIdx", ai,
			"subtitleIdx", si,
		)

		tcParams := state.transcodeParams(qi, ai, si)
		if tcParams != nil {
			tcParams.Offset = resumeOffsetSeconds(currentPosition())
		}
		go func() {
			audioID := 0
			if ai >= 0 && ai < len(state.audioStreamIDs) {
				audioID = state.audioStreamIDs[ai]
			}
			subtitleID := 0
			if si >= 0 && si < len(state.subtitleStreamIDs) {
				subtitleID = state.subtitleStreamIDs[si]
			}

			selection := sources.StreamSelection{}
			if audioID > 0 {
				selection.AudioStreamID = &audioID
			}
			if len(state.subtitleStreamIDs) > 1 {
				selection.SubtitleStreamID = &subtitleID
			}

			if state.partID > 0 {
				if err := state.source.SelectStreams(state.ctx, state.partID, selection); err != nil {
					slog.Warn("player: saving stream selection failed", "error", err)
				}
			}

			if tcParams == nil {
				schwifty.OnMainThreadOncePure(func() {
					if onAudioChanged != nil {
						onAudioChanged(ai)
					}
					if onSubtitleChanged != nil {
						onSubtitleChanged(si - 1)
					}
					onChanged(state.source.StreamURL(state.params.PartKey), nil)
				})
				return
			}

			q := state.source.BuildTranscodeQuery(*tcParams)
			startURL := state.source.TranscodeStartURL(q)
			if err := state.source.MakeTranscodeDecision(state.ctx, q); err != nil {
				slog.Error("player: decision call failed", "error", err)
				return
			}
			schwifty.OnMainThreadOncePure(func() {
				onChanged(startURL, tcParams)
			})
		}()
	}

	// fireSubtitleChange handles subtitle dropdown changes without restarting
	// the stream. It calls into the player's text-stream selector (which maps
	// to playbin3's select-streams event) and persists the selection to Plex
	// so other clients pick it up.
	//
	// Closing the outer popover here is deliberate: when GtkDropDown opens
	// its internal popover for the picker, GTK's nested-popover grab handling
	// leaves the parent settings popover in a state where click-outside no
	// longer dismisses it (only ESC does). Calling Popdown() on selection
	// matches the quality/audio behaviour and side-steps the stuck-popover
	// bug entirely.
	//
	// Dropdown index 0 is "None" (textStreamIdx = -1 disables text rendering);
	// indices 1+ correspond to the Nth text stream playbin3 sees, which lines
	// up with the order Plex returns the subtitle metadata.
	fireSubtitleChange := func() {
		rawPopover.Popdown()

		si := selectedDropDownIndex(subtitleDD)
		if si < 0 {
			si = 0
		}
		subtitleID := 0
		if si >= 0 && si < len(state.subtitleStreamIDs) {
			subtitleID = state.subtitleStreamIDs[si]
		}

		if onSubtitleChanged != nil {
			onSubtitleChanged(si - 1)
		}

		if len(state.subtitleStreamIDs) > 1 && state.partID > 0 {
			selection := sources.StreamSelection{SubtitleStreamID: &subtitleID}
			go func() {
				if err := state.source.SelectStreams(state.ctx, state.partID, selection); err != nil {
					slog.Warn("player: saving subtitle selection failed", "error", err)
				}
			}()
		}
	}

	qualityDD.ConnectSignal("notify::selected", new(func() {
		fireStreamSwitch()
	}))
	audioDD.ConnectSignal("notify::selected", new(func() {
		fireStreamSwitch()
	}))
	subtitleDD.ConnectSignal("notify::selected", new(func() {
		// In direct play, playbin3 changes the text stream natively (cheap).
		// In transcoded play, the active session was started with a specific
		// SubtitleStreamID — Plex burns/muxes that subtitle server-side, so
		// switching subtitles requires restarting the stream with the new pick.
		// External/sidecar subtitles also need a managed stream because they
		// don't exist in the direct-play media container.
		si := selectedDropDownIndex(subtitleDD)
		if si < 0 {
			si = 0
		}
		if (transcodeActive != nil && transcodeActive()) || state.subtitleRequiresManagedPlayback(si) {
			fireStreamSwitch()
		} else {
			fireSubtitleChange()
		}
	}))

	initialAudioIdx := -1
	if len(audioLabels) > 0 {
		initialAudioIdx = int(selectedAudio)
	}
	return rawPopover, initialStreamSelection{
		AudioIdx:    initialAudioIdx,
		SubtitleIdx: int(selectedSubtitle) - 1,
	}
}
