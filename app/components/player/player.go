package player

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync/atomic"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/google/uuid"
	"github.com/jwijenbergh/puregotk/v4/gdk"
	"github.com/jwijenbergh/puregotk/v4/gio"
	"github.com/jwijenbergh/puregotk/v4/glib"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

// PlayerParams configures a new player window.
type PlayerParams struct {
	Title     string
	PartKey   string // raw media part key (e.g. "/library/parts/12345/file.mkv")
	Window    *gtk.Window
	RatingKey string         // metadata ratingKey
	Media     []sources.Media // full Media array from Metadata
	Source    sources.Source  // the source for this playback
}

// NewPlayer creates a fullscreen video player window with overlay controls.
func NewPlayer(params PlayerParams) {
	src := params.Source
	sessionID := uuid.NewString()
	ctx, ctxCancel := context.WithCancel(context.Background())

	win := gtk.NewWindow()
	win.SetTitle(params.Title)
	win.SetTransientFor(params.Window)
	win.SetModal(true)

	// Create picture placeholder (media attached after decision resolves)
	picture := gtk.NewPicture()
	picture.SetContentFit(gtk.ContentFitCoverValue)
	picture.SetHexpand(true)
	picture.SetVexpand(true)
	picture.AddCssClass("scanline-player-picture")

	var media *gtk.MediaFile

	// Build overlay UI
	overlay := gtk.NewOverlay()
	overlay.SetChild(&picture.Widget)

	// Add CSS provider for black background (covers letterbox areas)
	// Use unique class names to avoid affecting other windows
	cssProvider := gtk.NewCssProvider()
	cssProvider.LoadFromString(`
		.scanline-player-window { background-color: #000000; background: #000000; }
		.scanline-player-overlay { background-color: #000000; background: #000000; }
		.scanline-player-picture { background-color: #000000; background: #000000; }
	`)
	display := gdk.DisplayGetDefault()
	gtk.StyleContextAddProviderForDisplay(
		display,
		cssProvider,
		uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION),
	)
	win.AddCssClass("scanline-player-window")
	overlay.AddCssClass("scanline-player-overlay")

	// --- Progress reporting ---
	var lastProgressUpdate atomic.Int64 // monotonic ms of last progress report

	sendProgress := func(state sources.PlaybackState) {
		if media == nil {
			return
		}
		ts := media.GetTimestamp()
		dur := media.GetDuration()
		if dur <= 0 {
			return
		}
		// GTK uses microseconds; Plex uses milliseconds
		timeMs := int(ts / 1000)
		durationMs := int(dur / 1000)
		rk := params.RatingKey
		go func() {
			if err := src.UpdateProgress(context.Background(), rk, state, timeMs, durationMs); err != nil {
				slog.Error("failed to update progress", "error", err)
			}
		}()
	}

	// --- Close button (top-right) ---
	closeBtnWidget := Button().
		IconName("window-close-symbolic").
		TooltipText("Close player").
		WithCSSClass("circular").
		WithCSSClass("osd").
		HAlign(gtk.AlignEndValue).
		VAlign(gtk.AlignStartValue).
		MarginTop(12).MarginEnd(12).
		ConnectClicked(func(b gtk.Button) {
			if media != nil {
				media.Pause()
			}
			win.Close()
		}).ToGTK()
	overlay.AddOverlay(closeBtnWidget)

	// --- Center playback controls ---
	var playing atomic.Bool
	var playPauseBtn *gtk.Button
	var currentTranscodeParams *sources.TranscodeParams // nil = direct play

	// doSeek handles seeking in both direct play and transcoded modes
	doSeek := func(targetMicroseconds int64) {
		if media == nil {
			return
		}

		if currentTranscodeParams == nil {
			// Direct play - simple seek
			media.Seek(targetMicroseconds)
			return
		}

		// Transcoded mode - restart stream at new position
		offsetSeconds := int(targetMicroseconds / 1000000)
		params := *currentTranscodeParams
		params.Offset = offsetSeconds

		go func() {
			q := src.BuildTranscodeQuery(params)
			if err := src.MakeTranscodeDecision(ctx, q); err != nil {
				slog.Error("player: seek decision failed", "error", err)
				return
			}
			newURL := src.TranscodeStartURL(q)
			schwifty.OnMainThreadOncePure(func() {
				vol := media.GetVolume()
				media.Pause()

				newFile := gio.FileNewForUri(newURL)
				newMedia := gtk.NewMediaFileForFile(newFile)
				newMedia.SetMuted(false)
				newMedia.SetVolume(vol)

				picture.SetPaintable(&gdk.PaintableBase{Ptr: newMedia.GoPointer()})
				media = newMedia
				newMedia.Play()
				playing.Store(true)
				if playPauseBtn != nil {
					playPauseBtn.SetIconName("media-playback-pause-symbolic")
				}
			})
		}()
	}

	skipBackBtn := Button().
		IconName("media-skip-backward-symbolic").
		TooltipText("Skip back 30 seconds").
		WithCSSClass("circular").
		WithCSSClass("osd").
		CSS("button { min-width: 48px; min-height: 48px; }").
		ConnectClicked(func(b gtk.Button) {
			if media == nil {
				return
			}
			ts := media.GetTimestamp()
			newTS := max(
				// 30 seconds in microseconds
				ts-30*1000000, 0)
			doSeek(newTS)
		})

	playPauseSchwifty := Button().
		IconName("media-playback-pause-symbolic").
		TooltipText("Play/Pause").
		WithCSSClass("circular").
		WithCSSClass("osd").
		CSS("button { min-width: 56px; min-height: 56px; }").
		ConnectConstruct(func(b *gtk.Button) {
			playPauseBtn = b
		}).
		ConnectClicked(func(b gtk.Button) {
			if media == nil {
				return
			}
			if playing.Load() {
				media.Pause()
				playing.Store(false)
				b.SetIconName("media-playback-start-symbolic")
				sendProgress(sources.StatePaused)
			} else {
				media.Play()
				playing.Store(true)
				b.SetIconName("media-playback-pause-symbolic")
				sendProgress(sources.StatePlaying)
			}
		})

	skipFwdBtn := Button().
		IconName("media-skip-forward-symbolic").
		TooltipText("Skip forward 30 seconds").
		WithCSSClass("circular").
		WithCSSClass("osd").
		CSS("button { min-width: 48px; min-height: 48px; }").
		ConnectClicked(func(b gtk.Button) {
			if media == nil {
				return
			}
			ts := media.GetTimestamp()
			dur := media.GetDuration()
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
	overlay.AddOverlay(centerControlsWidget)

	// --- Bottom bar ---
	var progressScale *gtk.Scale
	var seeking atomic.Bool

	titleLabel := Label(params.Title).
		WithCSSClass("heading").
		HAlign(gtk.AlignStartValue).
		HExpand(true).
		CSS("label { color: white; text-shadow: 0 1px 3px rgba(0,0,0,0.8); }")

	// Volume button with popover
	volumeScale := Scale(gtk.OrientationVerticalValue).
		Range(0, 1.0).
		Value(1.0).
		Inverted(true).
		SizeRequest(-1, 120).
		ConnectChangeValue(func(r gtk.Range, st gtk.ScrollType, val float64) bool {
			if media == nil {
				return false
			}
			if val < 0 {
				val = 0
			}
			if val > 1 {
				val = 1
			}
			media.SetVolume(val)
			return false
		})

	volumePopover := Popover(volumeScale).
		SizeRequest(40, 140)

	volumeBtn := MenuButton().
		IconName("audio-volume-high-symbolic").
		TooltipText("Adjust volume").
		WithCSSClass("circular").
		WithCSSClass("osd").
		Popover(volumePopover)

	// --- Settings popover (quality, audio, subtitles) ---
	var settingsPopover *gtk.Popover
	if len(params.Media) > 0 && len(params.Media[0].Part) > 0 {
		settingsPopover = buildSettingsPopover(params, src, sessionID, func(newURL string, transcodeParams *sources.TranscodeParams) {
			currentTranscodeParams = transcodeParams // Track current transcode state
			slog.Debug("player: switching stream", "url", newURL, "transcoding", transcodeParams != nil)

			var seekPos int64
			var vol float64
			if media != nil {
				seekPos = media.GetTimestamp()
				vol = media.GetVolume()
				media.Pause()
			} else {
				vol = 1.0
			}

			newFile := gio.FileNewForUri(newURL)
			newMedia := gtk.NewMediaFileForFile(newFile)
			newMedia.SetMuted(false)
			newMedia.SetVolume(vol)

			picture.SetPaintable(&gdk.PaintableBase{Ptr: newMedia.GoPointer()})
			media = newMedia

			newMedia.Play()
			playing.Store(true)
			if playPauseBtn != nil {
				playPauseBtn.SetIconName("media-playback-pause-symbolic")
			}

			// Poll until stream is prepared, then seek to saved position (only for direct play)
			if seekPos > 0 && transcodeParams == nil {
				seekCb := glib.SourceFunc(func(uintptr) bool {
					if err := media.GetError(); err != nil {
						slog.Error("player: stream error after switch", "error", err.Error())
						return false
					}
					if !media.IsPrepared() {
						return true // keep polling
					}
					media.Seek(seekPos)
					return false
				})
				glib.TimeoutAdd(200, &seekCb, 0)
			}
		})
	}

	settingsBtn := MenuButton().
		IconName("emblem-system-symbolic").
		TooltipText("Playback settings").
		WithCSSClass("circular").
		WithCSSClass("osd")

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
			if media == nil {
				return false
			}
			dur := media.GetDuration()
			if dur > 0 {
				seeking.Store(true)
				doSeek(int64(val))
				seeking.Store(false)
			}
			return false
		})

	var currentTimeLabel, remainingTimeLabel *gtk.Label
	timeCSS := "label { color: white; font-size: 12px; text-shadow: 0 1px 2px rgba(0,0,0,0.8); }"

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
		Spacing(4).
		VAlign(gtk.AlignEndValue).
		HExpand(true).
		MarginBottom(12).
		WithCSSClass("osd").
		CSS("box { background: linear-gradient(transparent, rgba(0,0,0,0.7)); padding: 12px 0; }").
		ToGTK()

	overlay.AddOverlay(bottomBarWidget)

	// --- Controls visibility (auto-hide) ---
	controlWidgets := []*gtk.Widget{closeBtnWidget, centerControlsWidget, bottomBarWidget}
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
			hideTimerID.Store(uint32(id))
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
	overlay.AddController(&motionCtrl.EventController)

	// --- Prevent auto-hide while settings popover is open ---
	if settingsPopover != nil {
		mapCb := func(w gtk.Widget) {
			if old := hideTimerID.Load(); old != 0 {
				glib.SourceRemove(uint(old))
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

	// --- ESC key to close ---
	keyCtrl := gtk.NewEventControllerKey()
	keyPressedCb := func(ctrl gtk.EventControllerKey, keyval uint, keycode uint, state gdk.ModifierType) bool {
		switch keyval {
		case uint(gdk.KEY_Escape):
			win.Close()
			return true
		case uint(gdk.KEY_space):
			if media == nil {
				return true
			}
			if playing.Load() {
				media.Pause()
				playing.Store(false)
				if playPauseBtn != nil {
					playPauseBtn.SetIconName("media-playback-start-symbolic")
				}
				sendProgress(sources.StatePaused)
			} else {
				media.Play()
				playing.Store(true)
				if playPauseBtn != nil {
					playPauseBtn.SetIconName("media-playback-pause-symbolic")
				}
				sendProgress(sources.StatePlaying)
			}
			return true
		case uint(gdk.KEY_Left):
			if media == nil {
				return true
			}
			ts := media.GetTimestamp()
			newTS := max(ts-30*1000000, 0)
			doSeek(newTS)
			return true
		case uint(gdk.KEY_Right):
			if media == nil {
				return true
			}
			ts := media.GetTimestamp()
			dur := media.GetDuration()
			newTS := ts + 30*1000000
			if dur > 0 && newTS > dur {
				newTS = dur
			}
			doSeek(newTS)
			return true
		case uint(gdk.KEY_Up):
			if media == nil {
				return true
			}
			vol := media.GetVolume() + 0.05
			if vol > 1 {
				vol = 1
			}
			media.SetVolume(vol)
			return true
		case uint(gdk.KEY_Down):
			if media == nil {
				return true
			}
			vol := media.GetVolume() - 0.05
			if vol < 0 {
				vol = 0
			}
			media.SetVolume(vol)
			return true
		}
		return false
	}
	keyCtrl.ConnectKeyPressed(&keyPressedCb)

	// --- Position ticker ---
	var tickerID atomic.Uint32
	var audioLogged atomic.Bool
	tickerCb := glib.SourceFunc(func(uintptr) bool {
		if media == nil {
			return true // keep polling, media not ready yet
		}
		// Log audio diagnostics once when media is prepared
		if media.IsPrepared() && !audioLogged.Swap(true) {
			slog.Debug("player: media state",
				"hasAudio", media.HasAudio(),
				"muted", media.GetMuted(),
				"volume", media.GetVolume(),
			)
		}
		dur := media.GetDuration()
		ts := media.GetTimestamp()
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
					if err := src.UpdateProgress(context.Background(), rk, sources.StatePlaying, timeMs, durationMs); err != nil {
						slog.Error("failed to update progress", "error", err)
					}
				}()
			}
		}
		// Update play/pause icon if stream ended
		if media.GetEnded() && playPauseBtn != nil {
			playing.Store(false)
			playPauseBtn.SetIconName("media-playback-start-symbolic")
		}
		return true // G_SOURCE_CONTINUE
	})
	tid := glib.TimeoutAdd(500, &tickerCb, 0)
	tickerID.Store(uint32(tid))

	// --- Set up window ---
	win.SetChild(&overlay.Widget)
	win.AddController(&keyCtrl.EventController)

	closeRequestCb := func(w gtk.Window) bool {
		ctxCancel()
		if media != nil {
			// Report stopped state to server
			sendProgress(sources.StateStopped)
			// Scrobble if >90% watched
			dur := media.GetDuration()
			ts := media.GetTimestamp()
			if dur > 0 && ts > 0 && float64(ts)/float64(dur) > 0.9 {
				go src.Scrobble(context.Background(), params.RatingKey) //nolint:errcheck // fire-and-forget
			}
			media.Pause()
		}
		if id := tickerID.Load(); id != 0 {
			glib.SourceRemove(uint(id))
			tickerID.Store(0)
		}
		if id := hideTimerID.Load(); id != 0 {
			glib.SourceRemove(uint(id))
			hideTimerID.Store(0)
		}
		// Remove CSS provider to avoid affecting other windows
		gtk.StyleContextRemoveProviderForDisplay(display, cssProvider)
		win.Destroy()
		return true
	}
	win.ConnectCloseRequest(&closeRequestCb)

	win.Fullscreen()
	win.Present()

	// Resolve playback URL via decision endpoint, then start playback
	go func() {
		streamURL := src.ResolvePlaybackURL(ctx, params.PartKey, params.RatingKey, sessionID)
		schwifty.OnMainThreadOncePure(func() {
			gioFile := gio.FileNewForUri(streamURL)
			media = gtk.NewMediaFileForFile(gioFile)
			media.SetMuted(false)
			media.SetVolume(1.0)
			picture.SetPaintable(&gdk.PaintableBase{Ptr: media.GoPointer()})
			media.Play()
			playing.Store(true)
			if playPauseBtn != nil {
				playPauseBtn.SetIconName("media-playback-pause-symbolic")
			}
			scheduleHide()
		})
	}()

	// Prevent GC from collecting closures that reference media
	runtime.KeepAlive(media)
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
