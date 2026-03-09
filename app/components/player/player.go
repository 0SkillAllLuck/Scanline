package player

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync/atomic"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/adw"
	"codeberg.org/puregotk/puregotk/v4/gdk"
	"codeberg.org/puregotk/puregotk/v4/gio"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/google/uuid"
)

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
}

// NewPlayer creates a video player with overlay controls.
// When the windowed player preference is enabled, the player reuses the main
// window instead of opening a separate fullscreen window.
func NewPlayer(params PlayerParams) {
	src := params.Source
	sessionID := uuid.NewString()
	ctx, ctxCancel := context.WithCancel(params.Ctx)

	windowed := preference.Experimental().EnableWindowedPlayer()

	// In fullscreen mode we create a separate modal window.
	// In windowed mode we reuse the existing main window.
	var win *gtk.Window
	// adwWin is used in windowed mode to call SetContent/GetContent,
	// since AdwApplicationWindow does not support SetChild/GetChild.
	var adwWin *adw.ApplicationWindow
	if !windowed {
		win = gtk.NewWindow()
		win.SetTitle(params.Title)
		win.SetTransientFor(params.Window)
		win.SetModal(true)
	} else {
		win = params.Window
		adwWin = &adw.ApplicationWindow{}
		adwWin.SetGoPointer(win.GoPointer())
	}

	// Hide parent window content so it can't show through the player.
	// For AdwApplicationWindow (windowed mode) use GetContent; for plain
	// gtk.Window (fullscreen mode) use GetChild.
	var parentContent *gtk.Widget
	if adwWin != nil {
		parentContent = adwWin.GetContent()
	} else {
		parentContent = params.Window.GetChild()
	}
	if parentContent != nil {
		parentContent.SetVisible(false)
	}

	// Create picture placeholder (media attached after decision resolves)
	picture := gtk.NewPicture()
	picture.SetContentFit(gtk.ContentFitContainValue)
	picture.SetHexpand(true)
	picture.SetVexpand(true)
	var media *gtk.MediaFile

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
	// In fullscreen mode it closes the separate window (triggering close-request).
	// In windowed mode it performs cleanup inline.
	var closePlayer func()

	// --- Top bar (fullscreen toggle + close) ---
	var fullscreenBtn *gtk.Button

	closeBtnSchwifty := Button().
		IconName("window-close-symbolic").
		TooltipText("Close player").
		WithCSSClass("circular").
		CSS(controlBtnCSS).
		ConnectClicked(func(b gtk.Button) {
			if media != nil {
				media.Pause()
			}
			closePlayer()
		})

	var topBarWidget *gtk.Widget
	if windowed {
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
		topBarWidget = HStack(fullscreenToggle, Spacer(), closeBtnSchwifty).
			HMargin(12).MarginTop(12).
			HAlign(gtk.AlignFillValue).
			VAlign(gtk.AlignStartValue).
			ToGTK()
	} else {
		topBarWidget = HStack(Spacer(), closeBtnSchwifty).
			HMargin(12).MarginTop(12).
			HAlign(gtk.AlignFillValue).
			VAlign(gtk.AlignStartValue).
			ToGTK()
	}

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

	centerBtnCSS := `button { background: transparent; border: none; box-shadow: none; min-width: 48px; min-height: 48px; border-radius: 9999px; color: white; }
		button:hover { background: rgba(255,255,255,0.15); }
		button image { -gtk-icon-shadow: 0 1px 4px rgba(0,0,0,0.9); -gtk-icon-size: 32px; }`

	skipBackBtn := Button().
		IconName("media-seek-backward-symbolic").
		TooltipText("Skip back 30 seconds").
		WithCSSClass("circular").
		CSS(centerBtnCSS).
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
		CSS(`button { background: transparent; border: none; box-shadow: none; min-width: 56px; min-height: 56px; border-radius: 9999px; color: white; }
			button:hover { background: rgba(255,255,255,0.15); }
			button image { -gtk-icon-shadow: 0 1px 4px rgba(0,0,0,0.9); -gtk-icon-size: 48px; }`).
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
		IconName("media-seek-forward-symbolic").
		TooltipText("Skip forward 30 seconds").
		WithCSSClass("circular").
		CSS(centerBtnCSS).
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
			if windowed {
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
			}
			return true
		case uint32(gdk.KEY_space):
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
		case uint32(gdk.KEY_Left):
			if media == nil {
				return true
			}
			ts := media.GetTimestamp()
			newTS := max(ts-30*1000000, 0)
			doSeek(newTS)
			return true
		case uint32(gdk.KEY_Right):
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
		case uint32(gdk.KEY_Up):
			if media == nil {
				return true
			}
			vol := media.GetVolume() + 0.05
			if vol > 1 {
				vol = 1
			}
			media.SetVolume(vol)
			return true
		case uint32(gdk.KEY_Down):
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
					if err := src.UpdateProgress(ctx, rk, sources.StatePlaying, timeMs, durationMs); err != nil {
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
	tickerID.Store(tid)

	// --- Set up window ---
	overlayWidget := Overlay(&picture.Widget).
		AddOverlay(topBarWidget).
		AddOverlay(centerControlsWidget).
		AddOverlay(bottomBarWidget).
		Controller(&motionCtrl.EventController).
		ToGTK()
	offload := gtk.NewGraphicsOffload(overlayWidget)
	offload.SetBlackBackground(true)
	if adwWin != nil {
		// Wrap in a WindowHandle so the window remains draggable even
		// though the header bar has been replaced by the player overlay.
		handle := gtk.NewWindowHandle()
		handle.SetChild(&offload.Widget)
		adwWin.SetContent(&handle.Widget)
	} else {
		win.SetChild(&offload.Widget)
	}
	win.AddController(&keyCtrl.EventController)

	// cleanup performs common teardown for both modes.
	cleanup := func() {
		ctxCancel()
		if media != nil {
			sendProgress(sources.StateStopped)
			dur := media.GetDuration()
			ts := media.GetTimestamp()
			if dur > 0 && ts > 0 && float64(ts)/float64(dur) > 0.9 {
				go src.Scrobble(ctx, params.RatingKey) //nolint:errcheck // fire-and-forget
			}
			media.Pause()
		}
		if id := tickerID.Load(); id != 0 {
			glib.SourceRemove(id)
			tickerID.Store(0)
		}
		if id := hideTimerID.Load(); id != 0 {
			glib.SourceRemove(id)
			hideTimerID.Store(0)
		}
	}

	if windowed {
		// Windowed mode: player lives inside the main window.
		closePlayer = func() {
			cleanup()
			// Remove controllers we added from the widgets they were attached to.
			win.RemoveController(&keyCtrl.EventController)
			overlayWidget.RemoveController(&motionCtrl.EventController)
			// Exit fullscreen if we toggled it
			if win.IsFullscreen() {
				win.Unfullscreen()
			}
			// Restore the original content
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
	} else {
		// Fullscreen mode: separate modal window.
		closePlayer = func() {
			win.Close()
		}

		closeRequestCb := func(w gtk.Window) bool {
			cleanup()
			if parentContent != nil {
				parentContent.SetVisible(true)
			}
			win.Destroy()
			router.Refresh()
			return true
		}
		win.ConnectCloseRequest(&closeRequestCb)

		win.Fullscreen()
		win.Present()
	}

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

			// Resume from saved position if ViewOffset is set
			if params.ViewOffset > 0 {
				targetUs := int64(params.ViewOffset) * 1000 // ms to µs
				seekCb := glib.SourceFunc(func(uintptr) bool {
					if err := media.GetError(); err != nil {
						slog.Error("player: stream error during resume seek", "error", err.Error())
						return false
					}
					if !media.IsPrepared() {
						return true // keep polling
					}
					doSeek(targetUs)
					return false
				})
				glib.TimeoutAdd(200, &seekCb, 0)
			}
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
