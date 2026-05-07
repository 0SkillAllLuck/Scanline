package player

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/adw"
	"codeberg.org/puregotk/puregotk/v4/gdk"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/go-gst/go-gst/gst"
	"github.com/google/uuid"
)

// NextEpisodeInfo contains pre-resolved information about the next episode.
type NextEpisodeInfo struct {
	Title      string
	PartKey    string
	RatingKey  string
	Media      []sources.Media
	ViewOffset int
	// Metadata is the full metadata of this next episode, used to resolve
	// the episode after it for chaining.
	Metadata *sources.Metadata
}

// PlayerParams configures a new player window.
type PlayerParams struct {
	Ctx        context.Context
	Title      string
	PartKey    string // raw media part key (e.g. "/library/parts/12345/file.mkv")
	Window     *gtk.Window
	RatingKey  string          // metadata ratingKey
	Media      []sources.Media // full Media array from Metadata
	Source     sources.Source  // the source for this playback
	ViewOffset int             // resume position in milliseconds

	// Parent / grandparent ratingKeys. For episodes: the season and show.
	// Empty for movies. Used by post-playback cache invalidation to clear
	// the season's children listing.
	ParentRatingKey      string
	GrandparentRatingKey string

	// NextEpisode is the pre-resolved next episode (nil for movies or last episode).
	NextEpisode *NextEpisodeInfo
}

// PlayerParamsForMetadata builds PlayerParams for a metadata item that has at
// least one playable media part. Caller is responsible for verifying that
// meta.Media[0].Part[0] exists.
func PlayerParamsForMetadata(ctx context.Context, meta *sources.Metadata, src sources.Source, win *gtk.Window, nextEp *NextEpisodeInfo) PlayerParams {
	return PlayerParams{
		Ctx:                  ctx,
		Title:                meta.Title,
		PartKey:              meta.Media[0].Part[0].Key,
		Window:               win,
		RatingKey:            meta.RatingKey,
		ParentRatingKey:      meta.ParentRatingKey,
		GrandparentRatingKey: meta.GrandparentRatingKey,
		Media:                meta.Media,
		Source:               src,
		ViewOffset:           meta.ViewOffset,
		NextEpisode:          nextEp,
	}
}

// NewPlayer creates a video player with overlay controls.
// The player reuses the application's main window — there is no separate
// fullscreen modal. The user can still toggle fullscreen via the F/F11 key
// or the dedicated button in the top bar.
func NewPlayer(params PlayerParams) {
	src := params.Source
	sessionID := uuid.NewString()
	ctx, ctxCancel := context.WithCancel(params.Ctx)
	initialTranscodeParams := initialManagedPlaybackParams(params, sessionID)

	win := params.Window
	adwWin := &adw.ApplicationWindow{}
	adwWin.SetGoPointer(win.GoPointer())

	// Hide parent window content so it can't show through the player.
	parentContent := adwWin.GetContent()
	if parentContent != nil {
		parentContent.SetVisible(false)
	}

	// Create picture placeholder (paintable attached once core is built).
	picture := gtk.NewPicture()
	picture.SetContentFit(gtk.ContentFitContainValue)
	picture.SetHexpand(true)
	picture.SetVexpand(true)

	// Wrap the picture (not the overlay) in GraphicsOffload so on Wayland
	// GTK can hand the paintable's dmabuf textures directly to the
	// compositor. Wrapping the overlay instead drives GraphicsOffload down
	// the fast path with non-trivial children and hits snapshot-machinery
	// assertions — which was the SIGSEGV the earlier attempt hit.
	pictureOffload := gtk.NewGraphicsOffload(&picture.Widget)
	pictureOffload.SetBlackBackground(true)

	knownDurationUs := int64(0)
	if len(params.Media) > 0 {
		if params.Media[0].Duration > 0 {
			knownDurationUs = int64(params.Media[0].Duration) * 1000
		} else if len(params.Media[0].Part) > 0 && params.Media[0].Part[0].Duration > 0 {
			knownDurationUs = int64(params.Media[0].Part[0].Duration) * 1000
		}
	}

	// --- Lifecycle guard ---
	var closed atomic.Bool // set during cleanup; prevents late async mutations

	// --- Progress reporting ---
	var lastProgressUpdate atomic.Int64 // monotonic ms of last progress report

	// pcore holds the playbin3 wrapper; built synchronously before any URI is set.
	var pcore *core
	// resumeOffsetUs is set on URI swaps so the OnAsyncDone callback can apply
	// a one-shot seek (used by ViewOffset resume on the initial direct-play URI).
	var resumeOffsetUs atomic.Int64

	currentTimestampUs := func() int64 {
		if pcore == nil {
			return 0
		}
		return pcore.PositionUs()
	}

	currentDurationUs := func() int64 {
		if pcore == nil {
			return knownDurationUs
		}
		return pcore.DurationUs()
	}

	sendProgress := func(state sources.PlaybackState) {
		if pcore == nil {
			return
		}
		ts := currentTimestampUs()
		dur := currentDurationUs()
		if dur <= 0 {
			return
		}
		// playbin3 reports microseconds (via core); Plex wants milliseconds.
		timeMs := int(ts / 1000)
		durationMs := int(dur / 1000)
		rk := params.RatingKey
		go func() {
			if err := src.UpdateProgress(ctx, rk, state, timeMs, durationMs); err != nil {
				slog.Error("failed to update progress", "error", err)
			}
		}()
	}

	// CSS for player control buttons: transparent by default, circular background on hover.
	controlBtnCSS := `button { background: transparent; border: none; box-shadow: none; min-width: 48px; min-height: 48px; border-radius: 9999px; color: white; }
		button:hover { background: rgba(255,255,255,0.15); }
		button image { -gtk-icon-shadow: 0 1px 3px rgba(0,0,0,0.8); -gtk-icon-size: 24px; }`

	// closePlayer tears down the player and restores the original window content.
	var closePlayer func()

	// --- Top bar (fullscreen toggle + close) ---
	var fullscreenBtn *gtk.Button

	closeBtnSchwifty := Button().
		IconName("window-close-symbolic").
		TooltipText("Close player").
		WithCSSClass("circular").
		CSS(controlBtnCSS).
		ConnectClicked(func(b gtk.Button) {
			if pcore != nil {
				pcore.Pause()
			}
			if closePlayer != nil {
				closePlayer()
			}
		})

	fullscreenToggle := Button().
		IconName("view-fullscreen-symbolic").
		TooltipText("Toggle fullscreen").
		WithCSSClass("circular").
		CSS(controlBtnCSS).
		ConnectConstruct(func(b *gtk.Button) {
			fullscreenBtn = b
		}).
		ConnectClicked(func(b gtk.Button) {
			if win.IsFullscreen() {
				win.Unfullscreen()
				b.SetIconName("view-fullscreen-symbolic")
			} else {
				win.Fullscreen()
				b.SetIconName("view-restore-symbolic")
			}
		})
	topBarWidget := HStack(fullscreenToggle, Spacer(), closeBtnSchwifty).
		HMargin(12).MarginTop(12).
		HAlign(gtk.AlignFillValue).
		VAlign(gtk.AlignStartValue).
		ToGTK()

	// --- Center playback controls ---
	var playing atomic.Bool
	var playPauseBtn *gtk.Button
	var currentTranscodeParams *sources.TranscodeParams // nil = direct play

	togglePlayPause := func() {
		if pcore == nil {
			return
		}
		if playing.Load() {
			pcore.Pause()
			playing.Store(false)
			if playPauseBtn != nil {
				playPauseBtn.SetIconName("media-playback-start-symbolic")
			}
			sendProgress(sources.StatePaused)
		} else {
			pcore.Play()
			playing.Store(true)
			if playPauseBtn != nil {
				playPauseBtn.SetIconName("media-playback-pause-symbolic")
			}
			sendProgress(sources.StatePlaying)
		}
	}

	// doSeek handles seeking in both direct play and transcoded modes
	doSeek := func(targetMicroseconds int64) {
		if pcore == nil {
			return
		}

		if currentTranscodeParams == nil {
			// Direct play — let playbin3 do an accurate seek inside the same
			// stream. baseOffset stays at 0, so no offset bookkeeping needed.
			pcore.SeekUs(targetMicroseconds)
			return
		}

		// Transcoded mode — Plex transcode segments cannot be seeked inside,
		// so we restart the stream at the new offset and let core track the
		// new baseOffset.
		offsetSeconds := int(targetMicroseconds / 1000000)
		tcParams := *currentTranscodeParams
		tcParams.Offset = offsetSeconds

		go func() {
			if len(params.Media) > 0 && len(params.Media[0].Part) > 0 {
				partID := params.Media[0].Part[0].ID
				if partID > 0 {
					selection := sources.StreamSelection{}
					if tcParams.AudioStreamID > 0 {
						audioID := tcParams.AudioStreamID
						selection.AudioStreamID = &audioID
					}
					if tcParams.SubtitleSelectionExplicit {
						subtitleID := tcParams.SubtitleStreamID
						selection.SubtitleStreamID = &subtitleID
					}
					if err := src.SelectStreams(ctx, partID, selection); err != nil {
						slog.Warn("player: seek stream selection failed", "error", err)
					}
				}
			}

			q := src.BuildTranscodeQuery(tcParams)
			if err := src.MakeTranscodeDecision(ctx, q); err != nil {
				slog.Error("player: seek decision failed", "error", err)
				return
			}
			newURL := src.TranscodeStartURL(q)
			schwifty.OnMainThreadOncePure(func() {
				if closed.Load() {
					return
				}
				resumeOffsetUs.Store(0)
				pcore.SetURI(newURL, int64(tcParams.Offset)*1000000)
				playing.Store(true)
				if playPauseBtn != nil {
					playPauseBtn.SetIconName("media-playback-pause-symbolic")
				}
			})
		}()
	}

	centerBtnCSS := `button { background: transparent; border: none; box-shadow: none; min-width: 48px; min-height: 48px; border-radius: 9999px; color: white; }
		button:hover { background: rgba(255,255,255,0.15); }
		button image { -gtk-icon-shadow: 0 1px 4px rgba(0,0,0,0.9); -gtk-icon-size: 32px; }`

	skipBackBtn := Button().
		IconName("media-seek-backward-symbolic").
		TooltipText("Skip back 30 seconds").
		WithCSSClass("circular").
		CSS(centerBtnCSS).
		ConnectClicked(func(b gtk.Button) {
			if pcore == nil {
				return
			}
			ts := currentTimestampUs()
			newTS := max(
				// 30 seconds in microseconds
				ts-30*1000000, 0)
			doSeek(newTS)
		})

	playPauseSchwifty := Button().
		IconName("media-playback-pause-symbolic").
		TooltipText("Play/Pause").
		WithCSSClass("circular").
		CSS(`button { background: transparent; border: none; box-shadow: none; min-width: 56px; min-height: 56px; border-radius: 9999px; color: white; }
			button:hover { background: rgba(255,255,255,0.15); }
			button image { -gtk-icon-shadow: 0 1px 4px rgba(0,0,0,0.9); -gtk-icon-size: 48px; }`).
		ConnectConstruct(func(b *gtk.Button) {
			playPauseBtn = b
		}).
		ConnectClicked(func(b gtk.Button) {
			togglePlayPause()
		})

	skipFwdBtn := Button().
		IconName("media-seek-forward-symbolic").
		TooltipText("Skip forward 30 seconds").
		WithCSSClass("circular").
		CSS(centerBtnCSS).
		ConnectClicked(func(b gtk.Button) {
			if pcore == nil {
				return
			}
			ts := currentTimestampUs()
			dur := currentDurationUs()
			newTS := ts + 30*1000000 // 30 seconds in microseconds
			if dur > 0 && newTS > dur {
				newTS = dur
			}
			doSeek(newTS)
		})

	centerControlsWidget := HStack(
		skipBackBtn,
		playPauseSchwifty,
		skipFwdBtn,
	).Spacing(16).
		HAlign(gtk.AlignCenterValue).
		VAlign(gtk.AlignCenterValue).
		ToGTK()

	// --- Bottom bar ---
	var progressScale *gtk.Scale
	var seeking atomic.Bool

	titleLabel := Label(params.Title).
		HAlign(gtk.AlignStartValue).
		HExpand(true).
		CSS("label { color: white; font-size: 16px; font-weight: 500; text-shadow: 0 1px 3px rgba(0,0,0,0.8); }")

	// Volume button with popover
	volumeScale := Scale(gtk.OrientationVerticalValue).
		Range(0, 1.0).
		Value(1.0).
		Inverted(true).
		SizeRequest(-1, 120).
		ConnectChangeValue(func(r gtk.Range, st gtk.ScrollType, val float64) bool {
			if pcore == nil {
				return false
			}
			pcore.SetVolume(val)
			return false
		})

	volumePopover := Popover(volumeScale).
		SizeRequest(40, 140)

	menuBtnCSS := `menubutton button { background: transparent; border: none; box-shadow: none; min-width: 48px; min-height: 48px; border-radius: 9999px; }
		menubutton button:hover { background: rgba(255,255,255,0.15); }`

	volumeBtn := MenuButton().
		IconName("audio-volume-high-symbolic").
		TooltipText("Adjust volume").
		WithCSSClass("flat").
		WithCSSClass("osd").
		CSS(menuBtnCSS).
		Popover(volumePopover)

	// --- Settings popover (quality, audio, subtitles) ---
	var settingsPopover *gtk.Popover
	// initialStreams contains the dropdown's starting stream indices. We apply
	// them to pcore once that's built so playback matches the saved Plex
	// selection — see comment on buildSettingsPopover.
	initialStreams := initialStreamSelection{AudioIdx: -1, SubtitleIdx: -1}
	if len(params.Media) > 0 && len(params.Media[0].Part) > 0 {
		settingsPopover, initialStreams = buildSettingsPopover(
			params,
			src,
			sessionID,
			ctx,
			func() int64 {
				return currentTimestampUs()
			},
			func() bool {
				return currentTranscodeParams != nil
			},
			func(newURL string, transcodeParams *sources.TranscodeParams) {
				if closed.Load() || pcore == nil {
					return
				}
				currentTranscodeParams = transcodeParams // Track current transcode state
				slog.Debug("player: switching stream", "url", redactToken(newURL), "transcoding", transcodeParams != nil)

				// Capture absolute position so we can resume across the URI swap
				// (only meaningful for direct play; transcode resumes via Offset).
				resumePosUs := currentTimestampUs()
				pcore.Pause()

				var baseOffsetUs int64
				if transcodeParams != nil {
					baseOffsetUs = int64(transcodeParams.Offset) * 1000000
					resumeOffsetUs.Store(0)
					pcore.SelectTextStreamByIndex(-1)
				} else {
					baseOffsetUs = 0
					resumeOffsetUs.Store(resumePosUs)
				}
				pcore.SetURI(newURL, baseOffsetUs)
				playing.Store(true)
				if playPauseBtn != nil {
					playPauseBtn.SetIconName("media-playback-pause-symbolic")
				}
			},
			func(audioIdx int) {
				if pcore == nil {
					return
				}
				pcore.SelectAudioStreamByIndex(audioIdx)
			},
			func(textIdx int) {
				if pcore == nil {
					return
				}
				pcore.SelectTextStreamByIndex(textIdx)
			},
		)
	}

	settingsBtn := MenuButton().
		IconName("emblem-system-symbolic").
		TooltipText("Playback settings").
		WithCSSClass("flat").
		WithCSSClass("osd").
		CSS(menuBtnCSS)

	if settingsPopover != nil {
		settingsBtn = settingsBtn.Popover(settingsPopover)
	} else {
		settingsBtn = settingsBtn.Sensitive(false)
	}

	topRow := HStack(
		titleLabel,
		HStack(volumeBtn, settingsBtn).Spacing(4),
	).Spacing(8).HMargin(16)

	progressSchwifty := Scale(gtk.OrientationHorizontalValue).
		Range(0, 1).
		HExpand(true).
		HMargin(16).
		CSS(`scale { margin-top: 0; margin-bottom: 0; }
			scale trough { min-height: 6px; }
			scale highlight { min-height: 6px; }
			scale slider { min-width: 16px; min-height: 16px; }`).
		ConnectConstruct(func(s *gtk.Scale) {
			progressScale = s
		}).
		ConnectChangeValue(func(r gtk.Range, st gtk.ScrollType, val float64) bool {
			if pcore == nil {
				return false
			}
			dur := currentDurationUs()
			if dur > 0 {
				seeking.Store(true)
				doSeek(int64(val))
				seeking.Store(false)
			}
			return false
		})

	var currentTimeLabel, remainingTimeLabel *gtk.Label
	timeCSS := "label { color: white; font-size: 13px; font-weight: bold; text-shadow: 0 1px 2px rgba(0,0,0,0.8); }"

	currentTimeSchwifty := Label("0:00").
		HAlign(gtk.AlignStartValue).
		CSS(timeCSS).
		ConnectConstruct(func(l *gtk.Label) {
			currentTimeLabel = l
		})

	remainingTimeSchwifty := Label("0:00").
		HAlign(gtk.AlignEndValue).
		CSS(timeCSS).
		ConnectConstruct(func(l *gtk.Label) {
			remainingTimeLabel = l
		})

	timeRow := HStack(currentTimeSchwifty, Spacer(), remainingTimeSchwifty).
		HMargin(16)

	bottomBarWidget := VStack(topRow, progressSchwifty, timeRow).
		Spacing(0).
		VAlign(gtk.AlignEndValue).
		HExpand(true).
		CSS("box { background: linear-gradient(transparent, rgba(0,0,0,0.7)); padding: 4px 0 20px 0; }").
		ToGTK()

	// pillCSS is shared by the bottom-right floating action buttons
	// (next-episode, skip-credits, skip-intro).
	pillCSS := `button { background: rgba(0,0,0,0.6); border: 1px solid rgba(255,255,255,0.2); color: white; padding: 8px 16px; font-weight: 500; }
			button:hover { background: rgba(255,255,255,0.2); }`

	makePillButton := func(iconName, label string, marginBottom int32, onClick func()) *gtk.Widget {
		w := Button().
			Child(HStack(
				Image().FromIconName(iconName),
				Label(label),
			).Spacing(8)).
			WithCSSClass("pill").
			CSS(pillCSS).
			ConnectClicked(func(b gtk.Button) { onClick() }).
			ToGTK()
		w.SetHalign(gtk.AlignEndValue)
		w.SetValign(gtk.AlignEndValue)
		w.SetMarginEnd(16)
		w.SetMarginBottom(marginBottom)
		w.SetVisible(false)
		return w
	}

	// --- Credits / intro skip trackers ---
	credits := &markerTracker{
		name:     sources.MarkerTypeCredits,
		autoSkip: preference.Experimental().AutoSkipCredits,
	}
	intro := &markerTracker{
		name:     sources.MarkerTypeIntro,
		autoSkip: preference.Experimental().AutoSkipIntro,
	}
	skipCreditsWidget := makePillButton(
		"media-seek-forward-symbolic", "Skip Credits", 170,
		func() { credits.seekToEnd(doSeek) },
	)
	credits.setVisible = skipCreditsWidget.SetVisible
	skipIntroWidget := makePillButton(
		"media-seek-forward-symbolic", "Skip Intro", 170,
		func() { intro.seekToEnd(doSeek) },
	)
	intro.setVisible = skipIntroWidget.SetVisible

	// --- "Next Episode" button ---
	var nextEpisodeWidget *gtk.Widget
	var nextEpisodeShown atomic.Bool

	// playNextEpisode resolves the next-next episode, closes this player, and
	// starts the next one. Shared by the button click and auto-play on end.
	var playNextEpisode func()

	if params.NextEpisode != nil {
		nextInfo := params.NextEpisode

		playNextEpisode = func() {
			// Resolve the next-next episode before closing (context still alive).
			var nextNext *NextEpisodeInfo
			var parent, grandparent string
			if nextInfo.Metadata != nil {
				nextNext = ResolveNextEpisode(ctx, src, nextInfo.Metadata)
				parent = nextInfo.Metadata.ParentRatingKey
				grandparent = nextInfo.Metadata.GrandparentRatingKey
			}
			closePlayer()
			NewPlayer(PlayerParams{
				Ctx:                  params.Ctx,
				Title:                nextInfo.Title,
				PartKey:              nextInfo.PartKey,
				Window:               params.Window,
				RatingKey:            nextInfo.RatingKey,
				ParentRatingKey:      parent,
				GrandparentRatingKey: grandparent,
				Media:                nextInfo.Media,
				Source:               src,
				ViewOffset:           nextInfo.ViewOffset,
				NextEpisode:          nextNext,
			})
		}

		nextEpisodeWidget = makePillButton(
			"media-skip-forward-symbolic",
			fmt.Sprintf("Next: %s", nextInfo.Title),
			115,
			playNextEpisode,
		)
	}

	// --- Controls visibility (auto-hide) ---
	controlWidgets := []*gtk.Widget{topBarWidget, centerControlsWidget, bottomBarWidget}
	var hideTimerID atomic.Uint32
	var lastActivityMs atomic.Int64 // timestamp of last activity in milliseconds

	showControls := func() {
		for _, w := range controlWidgets {
			w.SetOpacity(1)
		}
	}

	hideControls := func() {
		for _, w := range controlWidgets {
			w.SetOpacity(0)
		}
	}

	// Use a single persistent timer to avoid exhausting purego's callback limit.
	// The timer checks if 3 seconds have passed since last activity.
	hideTimerCallback := glib.SourceFunc(func(uintptr) bool {
		now := glib.GetMonotonicTime() / 1000 // convert to ms
		lastActivity := lastActivityMs.Load()
		if now-lastActivity >= 3000 {
			hideControls()
			hideTimerID.Store(0)
			return false // G_SOURCE_REMOVE - stop timer
		}
		return true // G_SOURCE_CONTINUE - keep checking
	})

	scheduleHide := func() {
		lastActivityMs.Store(glib.GetMonotonicTime() / 1000)
		// Only start timer if not already running
		if hideTimerID.Load() == 0 {
			id := glib.TimeoutAdd(500, &hideTimerCallback, 0) // check every 500ms
			hideTimerID.Store(id)
		}
	}

	motionCtrl := gtk.NewEventControllerMotion()
	motionCb := func(ctrl gtk.EventControllerMotion, x, y float64) {
		showControls()
		scheduleHide()
	}
	motionCtrl.ConnectMotion(&motionCb)

	enterCb := func(ctrl gtk.EventControllerMotion, x, y float64) {
		showControls()
		scheduleHide()
	}
	motionCtrl.ConnectEnter(&enterCb)

	leaveCb := func(ctrl gtk.EventControllerMotion) {
		scheduleHide()
	}
	motionCtrl.ConnectLeave(&leaveCb)
	// --- Prevent auto-hide while settings popover is open ---
	if settingsPopover != nil {
		mapCb := func(w gtk.Widget) {
			if old := hideTimerID.Load(); old != 0 {
				glib.SourceRemove(old)
				hideTimerID.Store(0)
			}
			showControls()
		}
		settingsPopover.ConnectMap(&mapCb)

		unmapCb := func(w gtk.Widget) {
			scheduleHide()
		}
		settingsPopover.ConnectUnmap(&unmapCb)
	}

	// --- ESC key to close, F/F11 to toggle fullscreen ---
	keyCtrl := gtk.NewEventControllerKey()
	keyPressedCb := func(ctrl gtk.EventControllerKey, keyval uint32, keycode uint32, state gdk.ModifierType) bool {
		switch keyval {
		case uint32(gdk.KEY_Escape):
			closePlayer()
			return true
		case uint32(gdk.KEY_f), uint32(gdk.KEY_F), uint32(gdk.KEY_F11):
			if win.IsFullscreen() {
				win.Unfullscreen()
				if fullscreenBtn != nil {
					fullscreenBtn.SetIconName("view-fullscreen-symbolic")
				}
			} else {
				win.Fullscreen()
				if fullscreenBtn != nil {
					fullscreenBtn.SetIconName("view-restore-symbolic")
				}
			}
			return true
		case uint32(gdk.KEY_space):
			togglePlayPause()
			return true
		case uint32(gdk.KEY_Left):
			if pcore == nil {
				return true
			}
			ts := currentTimestampUs()
			newTS := max(ts-30*1000000, 0)
			doSeek(newTS)
			return true
		case uint32(gdk.KEY_Right):
			if pcore == nil {
				return true
			}
			ts := currentTimestampUs()
			dur := currentDurationUs()
			newTS := ts + 30*1000000
			if dur > 0 && newTS > dur {
				newTS = dur
			}
			doSeek(newTS)
			return true
		case uint32(gdk.KEY_Up):
			if pcore == nil {
				return true
			}
			pcore.SetVolume(pcore.Volume() + 0.05)
			return true
		case uint32(gdk.KEY_Down):
			if pcore == nil {
				return true
			}
			pcore.SetVolume(pcore.Volume() - 0.05)
			return true
		}
		return false
	}
	keyCtrl.ConnectKeyPressed(&keyPressedCb)

	// --- Position ticker ---
	// Drives the progress scale, time labels, periodic UpdateProgress, and the
	// next-episode button. End-of-stream is delivered separately by the core
	// via OnEOS, so the ticker only needs to handle position/UI updates.
	var tickerID atomic.Uint32
	tickerCb := glib.SourceFunc(func(uintptr) bool {
		if pcore == nil {
			return true // keep polling, core not built yet
		}
		dur := currentDurationUs()
		ts := currentTimestampUs()
		if !seeking.Load() && progressScale != nil && dur > 0 {
			progressScale.SetRange(0, float64(dur))
			progressScale.SetValue(float64(ts))
		}
		if currentTimeLabel != nil {
			currentTimeLabel.SetText(formatMicroseconds(ts))
		}
		if remainingTimeLabel != nil && dur > 0 {
			remaining := max(dur-ts, 0)
			remainingTimeLabel.SetText("-" + formatMicroseconds(remaining))
		}
		// Periodic progress reporting (every 10 seconds while playing)
		if playing.Load() && dur > 0 {
			nowMs := glib.GetMonotonicTime() / 1000
			if nowMs-lastProgressUpdate.Load() >= 10000 {
				lastProgressUpdate.Store(nowMs)
				timeMs := int(ts / 1000)
				durationMs := int(dur / 1000)
				rk := params.RatingKey
				go func() {
					if err := src.UpdateProgress(ctx, rk, sources.StatePlaying, timeMs, durationMs); err != nil {
						slog.Error("failed to update progress", "error", err)
					}
				}()
			}
		}
		// Show "Next Episode" button on the credits-start edge, falling back
		// to the 90% threshold when no credits marker was returned. One-shot:
		// once shown it stays for the rest of playback.
		if nextEpisodeWidget != nil && !nextEpisodeShown.Load() && dur > 0 && ts > 0 {
			creditsMs := credits.startMs.Load()
			show := false
			switch {
			case creditsMs > 0:
				show = ts >= creditsMs*1000
			case creditsMs < 0:
				show = float64(ts)/float64(dur) >= 0.9
			}
			if show && nextEpisodeShown.CompareAndSwap(false, true) {
				nextEpisodeWidget.SetVisible(true)
			}
		}
		credits.tick(ts, doSeek)
		intro.tick(ts, doSeek)
		return true // G_SOURCE_CONTINUE
	})
	tid := glib.TimeoutAdd(500, &tickerCb, 0)
	tickerID.Store(tid)

	// --- Set up window ---
	// The overlay base is the offload-wrapped picture, keeping the offload
	// widget's child a single paintable leaf. Controls layer above via
	// Overlay's secondary children; when the auto-hide timer drops them,
	// GTK can re-engage compositor offload.
	overlay := Overlay(&pictureOffload.Widget).
		AddOverlay(topBarWidget).
		AddOverlay(centerControlsWidget).
		AddOverlay(bottomBarWidget).
		AddOverlay(skipCreditsWidget).
		AddOverlay(skipIntroWidget)
	if nextEpisodeWidget != nil {
		overlay = overlay.AddOverlay(nextEpisodeWidget)
	}
	overlayWidget := overlay.
		Controller(&motionCtrl.EventController).
		ToGTK()
	// WindowHandle keeps the window draggable even though the header bar
	// has been replaced by the player overlay.
	handle := gtk.NewWindowHandle()
	handle.SetChild(overlayWidget)
	adwWin.SetContent(&handle.Widget)
	win.AddController(&keyCtrl.EventController)

	// cleanup tears down the player. Order matters: stop timers first so
	// the ticker can't race with pcore teardown; do the final progress
	// reports synchronously while ctx is still alive; detach the paintable
	// from the picture before NULL'ing the pipeline (otherwise snapshots
	// hit a closing sink); ctxCancel() last so blocking HTTP can finish.
	cleanup := func() bool {
		if !closed.CompareAndSwap(false, true) {
			return false
		}
		if id := tickerID.Load(); id != 0 {
			glib.SourceRemove(id)
			tickerID.Store(0)
		}
		if id := hideTimerID.Load(); id != 0 {
			glib.SourceRemove(id)
			hideTimerID.Store(0)
		}
		if pcore != nil {
			pcore.Pause()
			dur := currentDurationUs()
			ts := currentTimestampUs()
			progressReported := false
			if dur > 0 {
				timeMs := int(ts / 1000)
				durationMs := int(dur / 1000)
				if err := src.UpdateProgress(ctx, params.RatingKey, sources.StateStopped, timeMs, durationMs); err != nil {
					slog.Error("failed to send final progress", "error", err)
				} else {
					progressReported = true
				}
			}
			if dur > 0 && ts > 0 && float64(ts)/float64(dur) > 0.9 {
				if err := src.Scrobble(ctx, params.RatingKey); err != nil {
					slog.Error("failed to scrobble", "error", err)
				} else {
					progressReported = true
				}
			}
			if progressReported {
				src.InvalidateAfterPlayback(params.RatingKey, params.ParentRatingKey, params.GrandparentRatingKey)
			}
			// Detach paintable before pcore.Close — otherwise the picture
			// may snapshot a sink whose state is being freed (SIGSEGV in
			// gdk_paintable_snapshot). A typed-nil PaintableBase passes a
			// NULL pointer through to gtk_picture_set_paintable.
			picture.SetPaintable((*gdk.PaintableBase)(nil))
			pcore.Close()
		}
		ctxCancel()
		return true
	}

	// Build the playbin3 wrapper before presenting the window so the
	// picture already has a paintable when GTK measures content. The bus
	// watch dispatches on the GLib main loop, so callbacks may touch UI
	// state directly. paintableAttached guards a one-shot SetPaintable —
	// snapshotting gtk4paintablesink before its first preroll has SIGSEGV'd
	// in the past, so we wait for PAUSED before attaching.
	var paintableAttached atomic.Bool
	{
		var coreErr error
		pcore, coreErr = newCore(coreOptions{
			OnError: func(err error) {
				slog.Error("player: pipeline error", "error", err)
				if closed.Load() {
					return
				}
				if closePlayer != nil {
					closePlayer()
				}
			},
			OnEOS: func() {
				if closed.Load() {
					return
				}
				if playNextEpisode != nil {
					playNextEpisode()
					return
				}
				if closePlayer != nil {
					closePlayer()
				}
			},
			OnStateChange: func(state gst.State) {
				slog.Debug("player: pipeline state", "state", state.String())
				if pcore == nil || paintableAttached.Load() {
					return
				}
				if state == gst.StatePaused || state == gst.StatePlaying {
					if !paintableAttached.Swap(true) {
						picture.SetPaintable(pcore.Paintable())
					}
				}
			},
			OnAsyncDone: func() {
				if closed.Load() {
					return
				}
				if target := resumeOffsetUs.Swap(0); target > 0 {
					pcore.SeekUs(target)
				}
			},
			OnBuffering: func(percent int) {
				slog.Debug("player: buffering", "percent", percent)
			},
		})
		if coreErr != nil {
			slog.Error("player: failed to build playbin3 pipeline", "error", coreErr)
			cleanup()
			win.RemoveController(&keyCtrl.EventController)
			overlayWidget.RemoveController(&motionCtrl.EventController)
			adwWin.SetContent(parentContent)
			if parentContent != nil {
				parentContent.SetVisible(true)
			}
			router.Refresh()
			return
		}

		// pcore now owns a paintable wrapper whose finalizer is disabled
		// (see readPaintableProperty). If anything between here and the
		// closePlayer wiring panics, close pcore on the main thread before
		// re-raising so the paintable doesn't leak.
		defer func() {
			if r := recover(); r != nil {
				if pcore != nil {
					pcore.Close()
				}
				panic(r)
			}
		}()

		pcore.SetKnownDurationUs(knownDurationUs)
		if initialTranscodeParams == nil {
			// Sync playbin with the dropdown's initial values. Without this,
			// saved Plex audio/subtitle selections can disagree with the
			// container defaults that direct play chooses. The actual
			// select-streams event is deferred until StreamCollection arrives.
			pcore.SelectAudioStreamByIndex(initialStreams.AudioIdx)
			pcore.SelectTextStreamByIndex(initialStreams.SubtitleIdx)
		} else {
			// Server-managed subtitle playback burns/muxes the selected
			// subtitle into the transcode output, so there is no local text
			// stream to select.
			pcore.SelectTextStreamByIndex(-1)
		}
	}

	closePlayer = func() {
		if !cleanup() {
			return
		}
		// Remove controllers we added from the widgets they were attached to.
		win.RemoveController(&keyCtrl.EventController)
		overlayWidget.RemoveController(&motionCtrl.EventController)
		// Exit fullscreen if we toggled it.
		if win.IsFullscreen() {
			win.Unfullscreen()
		}
		// Restore the original content.
		adwWin.SetContent(parentContent)
		if parentContent != nil {
			parentContent.SetVisible(true)
		}
		router.Refresh()
	}
	if preference.Experimental().StartInFullscreen() {
		win.Fullscreen()
	}
	win.Present()

	// Resolve markers in the background. Drives the next-episode button
	// timing and the skip-credits / skip-intro buttons (with their auto-skip
	// counterparts).
	go func() {
		trackers := map[string]*markerTracker{
			sources.MarkerTypeCredits: credits,
			sources.MarkerTypeIntro:   intro,
		}
		markers, err := src.GetMarkers(ctx, params.RatingKey)
		if err != nil {
			slog.Debug("player: failed to fetch markers", "error", err)
			for _, t := range trackers {
				t.setNotFound()
			}
			return
		}
		found := make(map[string]bool, len(trackers))
		for _, m := range markers {
			t, ok := trackers[m.Type]
			if !ok || found[m.Type] {
				continue
			}
			t.setRange(m.StartTimeOffset, m.EndTimeOffset)
			found[m.Type] = true
			slog.Debug("player: resolved "+m.Type+" marker",
				"start_ms", m.StartTimeOffset,
				"end_ms", m.EndTimeOffset)
		}
		for typ, t := range trackers {
			if !found[typ] {
				t.setNotFound()
				slog.Debug("player: no " + typ + " marker found")
			}
		}
	}()

	// Resolve playback URL via decision endpoint, then start playback. Initial
	// playback uses direct play unless the saved subtitle selection is
	// server-managed (external/sidecar), in which case Plex has to prepare a
	// transcode session because playbin cannot discover that subtitle in the
	// media container.
	go func() {
		streamURL := ""
		if initialTranscodeParams != nil {
			q := src.BuildTranscodeQuery(*initialTranscodeParams)
			if err := src.MakeTranscodeDecision(ctx, q); err != nil {
				slog.Error("player: initial decision failed", "error", err)
				return
			}
			streamURL = src.TranscodeStartURL(q)
		} else {
			streamURL = src.ResolvePlaybackURL(ctx, params.PartKey, params.RatingKey, sessionID)
		}
		schwifty.OnMainThreadOncePure(func() {
			if closed.Load() {
				return
			}
			currentTranscodeParams = initialTranscodeParams
			baseOffsetUs := int64(0)
			if initialTranscodeParams != nil {
				baseOffsetUs = int64(initialTranscodeParams.Offset) * 1000000
				resumeOffsetUs.Store(0)
			} else if params.ViewOffset > 0 {
				// Defer the resume seek until ASYNC_DONE is observed by the
				// core (avoids the polled IsPrepared loop the gtk.MediaFile
				// path used to need).
				resumeOffsetUs.Store(int64(params.ViewOffset) * 1000)
			}
			pcore.SetURI(streamURL, baseOffsetUs)
			playing.Store(true)
			if playPauseBtn != nil {
				playPauseBtn.SetIconName("media-playback-pause-symbolic")
			}
			scheduleHide()
		})
	}()

}

func formatMicroseconds(us int64) string {
	totalSeconds := us / 1000000
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func initialManagedPlaybackParams(params PlayerParams, sessionID string) *sources.TranscodeParams {
	if len(params.Media) == 0 || len(params.Media[0].Part) == 0 {
		return nil
	}

	selectedAudioID := 0
	selectedSubtitleID := 0
	selectedSubtitleNeedsManagedPlayback := false
	for _, stream := range params.Media[0].Part[0].Stream {
		switch stream.StreamType {
		case sources.StreamTypeAudio:
			if stream.Selected {
				selectedAudioID = stream.ID
			}
		case sources.StreamTypeSubtitle:
			if stream.Selected && streamRequiresManagedPlayback(stream) {
				selectedSubtitleID = stream.ID
				selectedSubtitleNeedsManagedPlayback = true
			}
		}
	}

	if !selectedSubtitleNeedsManagedPlayback {
		return nil
	}

	return &sources.TranscodeParams{
		RatingKey:                 params.RatingKey,
		SessionID:                 sessionID,
		DirectStreamAudio:         true,
		AudioStreamID:             selectedAudioID,
		SubtitleStreamID:          selectedSubtitleID,
		SubtitleSelectionExplicit: true,
		Offset:                    params.ViewOffset / 1000,
	}
}

// redactToken scrubs Plex authentication tokens out of strings before they
// reach a log sink. Only the value of `X-Plex-Token` is redacted; other
// query params (rating key, session ID) stay visible for correlation.
func redactToken(s string) string {
	const key = "X-Plex-Token="
	const replacement = "REDACTED"
	if !strings.Contains(s, key) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	rest := s
	for {
		i := strings.Index(rest, key)
		if i < 0 {
			b.WriteString(rest)
			return b.String()
		}
		valStart := i + len(key)
		valEnd := valStart
		for valEnd < len(rest) && !isTokenTerminator(rest[valEnd]) {
			valEnd++
		}
		b.WriteString(rest[:valStart])
		b.WriteString(replacement)
		// Advance past the original token value; do NOT rescan `REDACTED`,
		// otherwise an empty original value would loop forever.
		rest = rest[valEnd:]
	}
}

// isTokenTerminator reports whether c marks the end of a Plex token value
// across the contexts redactToken is called from: URL query strings (`&`,
// `#`), shell-style log lines (whitespace), quoted error messages (`"`,
// `'`), and angle-bracket-wrapped log fragments (`<`, `>`). Errs on the
// side of stopping early — better to redact too little context than to
// gobble up surrounding non-secret text.
func isTokenTerminator(c byte) bool {
	switch c {
	case '&', '#', '"', '\'', '<', '>', ' ', '\t', '\n', '\r':
		return true
	}
	return false
}
