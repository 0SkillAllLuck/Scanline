package player

import (
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"codeberg.org/puregotk/puregotk/v4/gdk"
	"github.com/go-gst/go-glib/glib"
	"github.com/go-gst/go-gst/gst"
)

// playbin3 GstPlayFlags strings. We set this property via SetArg (which
// parses the +-separated names through gst_util_set_object_arg) because the
// property is a flags GType — a plain int can't be assigned via go-glib's
// SetProperty.
//
// https://gstreamer.freedesktop.org/documentation/playback/playbin3.html
const (
	playFlagsTextOn  = "video+audio+text+soft-volume+deinterlace+soft-colorbalance"
	playFlagsTextOff = "video+audio+soft-volume+deinterlace+soft-colorbalance"
)

var gstInitOnce sync.Once

func ensureGstInit() {
	// gst.Init is idempotent and returns silently on failure (it logs a
	// g_warning and continues). sync.Once is enough — there is nothing
	// useful we can return on failure that the subsequent NewElement call
	// won't already surface.
	gstInitOnce.Do(func() { gst.Init(nil) })
}

// streamInfo describes one selectable stream from the active StreamCollection.
type streamInfo struct {
	StreamID string
	Selected bool
	stream   *gst.Stream
}

type coreOptions struct {
	OnEOS         func()
	OnError       func(error)
	OnStateChange func(state gst.State)
	OnBuffering   func(percent int)
	OnAsyncDone   func()
}

// core wraps a playbin3 + gtk4paintablesink pipeline. One instance per playback.
//
// The bus watch dispatches messages on the GLib main loop (the GTK main thread),
// so callbacks may safely touch UI state without further marshalling.
type core struct {
	pipeline     *gst.Element
	pipelineName string
	// videoSink is the element actually plugged into playbin3's video-sink
	// property. It may be paintableSink directly or paintableSink wrapped in
	// glsinkbin when one is available.
	videoSink *gst.Element
	// paintableSink is the underlying gtk4paintablesink — the element we
	// read the GdkPaintable from. The sink itself is kept alive by the
	// pipeline (and by glsinkbin when wrapped); this Go-side field exists
	// purely so the wrapper's GC finalizer can't fire and unref the sink
	// out from under the picture. It is set to nil only in Close().
	paintableSink *gst.Element
	bus           *gst.Bus
	// paintableObj keeps a Go-side reference to the GdkPaintable returned by
	// gtk4paintablesink. Without this, the *glib.Object wrapper goes out of
	// scope after newCore returns; its finalizer unrefs the paintable, and
	// the bare uintptr in `paintable` is left dangling — which crashes inside
	// GTK once the picture starts asking for frames.
	paintableObj *glib.Object
	paintable    *gdk.PaintableBase

	opts coreOptions

	mu           sync.Mutex
	audioStreams []streamInfo
	textStreams  []streamInfo
	videoStreams []streamInfo
	// Pending indices remember selection calls that arrived before the
	// StreamCollection was visible. processCollection re-applies them once
	// streams are populated. Guarded by mu, alongside the stream lists.
	pendingAudioIdx    int
	pendingAudioActive bool
	pendingTextIdx     int
	pendingTextActive  bool

	closed          atomic.Bool
	asyncDone       atomic.Bool
	baseOffsetUs    atomic.Int64
	knownDurationUs atomic.Int64
	textEnabled     atomic.Bool
	// desiredPlaying reflects the user-facing intent: true after Play() /
	// SetURI(), false after Pause(). The MessageBuffering handler consults
	// this so a 100%-buffered message can't silently resume playback the
	// user paused mid-rebuffer.
	desiredPlaying atomic.Bool
}

func newCore(opts coreOptions) (*core, error) {
	ensureGstInit()

	pipeline, err := gst.NewElement("playbin3")
	if err != nil {
		return nil, fmt.Errorf("create playbin3: %w", err)
	}

	paintableSink, err := gst.NewElement("gtk4paintablesink")
	if err != nil {
		return nil, fmt.Errorf("gtk4paintablesink not available (install gst-plugins-rs): %w", err)
	}

	paintableObj, paintable, err := readPaintableProperty(paintableSink)
	if err != nil {
		return nil, err
	}

	// Wrap the paintable sink in glsinkbin so it gets a proper GL context
	// from playbin3 — this is the pattern recommended by gst-plugins-rs and
	// avoids a class of preroll/snapshot races in gtk4paintablesink.
	videoSink := paintableSink
	if glbin, glErr := gst.NewElement("glsinkbin"); glErr == nil {
		if err := glbin.SetProperty("sink", paintableSink); err != nil {
			slog.Warn("player core: glsinkbin set sink failed, falling back", "error", err)
		} else {
			videoSink = glbin
		}
	} else {
		slog.Debug("player core: glsinkbin unavailable, using bare paintable sink", "error", glErr)
	}

	if err := pipeline.SetProperty("video-sink", videoSink); err != nil {
		return nil, fmt.Errorf("set video-sink: %w", err)
	}

	// Match Tonearm's buffer tuning so first-frame latency and rebuffering
	// behave the same way for HTTP streams.
	_ = pipeline.SetProperty("buffer-size", 20*1024*1024)
	_ = pipeline.SetProperty("buffer-duration", (30 * time.Second).Nanoseconds())

	c := &core{
		pipeline:      pipeline,
		pipelineName:  pipeline.GetName(),
		videoSink:     videoSink,
		paintableSink: paintableSink,
		paintableObj:  paintableObj,
		paintable:     paintable,
		opts:          opts,
	}
	c.textEnabled.Store(true)
	c.applyFlags()

	c.bus = pipeline.GetBus()
	c.bus.AddWatch(c.onBusMessage)

	return c, nil
}

// readPaintableProperty pulls the GdkPaintable out of gtk4paintablesink and
// returns both the *glib.Object wrapper (so the caller can pin its lifetime)
// and a puregotk-typed PaintableBase pointing at the same C object.
//
// Lifetime note: go-glib's marshalObject calls glib.Take() on the returned
// GObject which (a) refs+sinks the object and (b) installs a Go finalizer
// that calls g_object_unref(). We must NOT let that finalizer run on its
// own — gtk4paintablesink's proxy paintable is bound to the GTK main
// context and unref'ing it from a finalizer goroutine corrupts the
// picture's snapshot path. We clear the finalizer here and rely on Close()
// to drop the ref on the main thread (where newCore was called).
func readPaintableProperty(sink *gst.Element) (*glib.Object, *gdk.PaintableBase, error) {
	val, err := sink.GetProperty("paintable")
	if err != nil {
		return nil, nil, fmt.Errorf("get paintable property: %w", err)
	}
	if val == nil {
		return nil, nil, fmt.Errorf("paintable property was nil")
	}
	obj, ok := val.(*glib.Object)
	if !ok {
		return nil, nil, fmt.Errorf("paintable property has unexpected type %T", val)
	}
	if obj.Native() == nil {
		return nil, nil, fmt.Errorf("paintable property has nil native pointer")
	}
	// Disable the auto-finalizer that glib.Take() installed; Close() unrefs
	// explicitly on the GTK main thread.
	runtime.SetFinalizer(obj, nil)
	return obj, &gdk.PaintableBase{Ptr: uintptr(obj.Native())}, nil
}

// Paintable returns a GdkPaintable suitable for gtk.Picture.SetPaintable.
// Stable until Close() is called; afterwards returns nil so that callers
// reattaching to a closed core get a typed-nil paintable rather than a
// dangling pointer.
func (c *core) Paintable() *gdk.PaintableBase {
	return c.paintable
}

// SetURI swaps the active source. baseOffsetUs is the wall-clock position
// represented by t=0 of the new stream — non-zero when Plex is transcoding
// from an offset, zero for direct play. The pipeline cycles to NULL first to
// release any previous stream's resources.
func (c *core) SetURI(uri string, baseOffsetUs int64) {
	if c.closed.Load() {
		return
	}

	c.asyncDone.Store(false)
	c.baseOffsetUs.Store(baseOffsetUs)

	c.mu.Lock()
	c.audioStreams = nil
	c.textStreams = nil
	c.videoStreams = nil
	c.mu.Unlock()

	if err := c.pipeline.SetState(gst.StateNull); err != nil {
		slog.Warn("player core: SetState(Null) failed", "error", err)
	}
	if err := c.pipeline.SetProperty("uri", uri); err != nil {
		slog.Error("player core: set uri failed", "error", err)
		return
	}
	c.applyFlags()
	c.desiredPlaying.Store(true)
	if err := c.pipeline.SetState(gst.StatePlaying); err != nil {
		slog.Warn("player core: SetState(Playing) failed", "error", err)
	}
}

// Play resumes playback after Pause. No-op before SetURI.
func (c *core) Play() {
	if c.closed.Load() {
		return
	}
	c.desiredPlaying.Store(true)
	if err := c.pipeline.SetState(gst.StatePlaying); err != nil {
		slog.Warn("player core: SetState(Playing) failed", "error", err)
	}
}

// Pause halts playback while keeping the current frame on the paintable.
func (c *core) Pause() {
	if c.closed.Load() {
		return
	}
	c.desiredPlaying.Store(false)
	if err := c.pipeline.SetState(gst.StatePaused); err != nil {
		slog.Warn("player core: SetState(Paused) failed", "error", err)
	}
}

// SeekUs jumps to a position in the current stream, in microseconds relative
// to the stream's t=0 (i.e. NOT including baseOffsetUs). Always issues an
// accurate seek — slider drags and resume-from-progress need frame-accurate
// landings, and the keyboard skip nudges are already coarse enough that
// keyframe snapping wouldn't be a noticeable speedup.
func (c *core) SeekUs(us int64) {
	if c.closed.Load() {
		return
	}
	if us < 0 {
		us = 0
	}
	flags := gst.SeekFlagFlush | gst.SeekFlagAccurate
	c.pipeline.SeekTime(time.Duration(us)*time.Microsecond, flags)
}

// PositionUs returns the absolute wall-clock position in microseconds, or
// the configured baseOffsetUs if the pipeline hasn't reported async-done yet
// for the current URI (i.e. QueryPosition would return stale or zero data).
func (c *core) PositionUs() int64 {
	if c.closed.Load() {
		return c.baseOffsetUs.Load()
	}
	if !c.asyncDone.Load() {
		return c.baseOffsetUs.Load()
	}
	ok, posNs := c.pipeline.QueryPosition(gst.FormatTime)
	if !ok || posNs < 0 {
		return c.baseOffsetUs.Load()
	}
	return c.baseOffsetUs.Load() + posNs/1000
}

// DurationUs returns the absolute duration of the underlying media in
// microseconds. Falls back to a known/metadata duration set by the caller
// (via SetKnownDurationUs) before the pipeline can answer the query.
func (c *core) DurationUs() int64 {
	known := c.knownDurationUs.Load()
	if c.closed.Load() {
		return known
	}
	ok, durNs := c.pipeline.QueryDuration(gst.FormatTime)
	if !ok || durNs <= 0 {
		return known
	}
	queried := c.baseOffsetUs.Load() + durNs/1000
	if known > queried {
		return known
	}
	return queried
}

// SetKnownDurationUs records a fallback duration from the source's metadata,
// used while the pipeline hasn't fully prerolled.
func (c *core) SetKnownDurationUs(us int64) {
	if us < 0 {
		us = 0
	}
	c.knownDurationUs.Store(us)
}

// SetVolume clamps `v` to [0, 1] and applies it to the playbin's volume
// property. Volume persists across SetURI.
func (c *core) SetVolume(v float64) {
	if c.closed.Load() {
		return
	}
	switch {
	case v < 0:
		v = 0
	case v > 1:
		v = 1
	}
	if err := c.pipeline.SetProperty("volume", v); err != nil {
		slog.Warn("player core: set volume failed", "error", err)
	}
}

// Volume returns the current playbin volume in [0, 1]. Returns 1.0 on any
// failure path (including closed) so keyboard volume nudges land on a
// sensible value rather than slamming to 0.
func (c *core) Volume() float64 {
	if c.closed.Load() {
		return 1.0
	}
	val, err := c.pipeline.GetProperty("volume")
	if err != nil {
		return 1.0
	}
	if v, ok := val.(float64); ok {
		return v
	}
	return 1.0
}

// SetTextEnabled toggles the GST_PLAY_FLAG_TEXT bit on playbin3's flags
// property. When false, no subtitle/text stream is rendered regardless of
// what's available in the container.
func (c *core) SetTextEnabled(enabled bool) {
	c.textEnabled.Store(enabled)
	c.applyFlags()
}

func (c *core) applyFlags() {
	if c.closed.Load() {
		return
	}
	flagsStr := playFlagsTextOff
	if c.textEnabled.Load() {
		flagsStr = playFlagsTextOn
	}
	c.pipeline.SetArg("flags", flagsStr)
}

// SelectTextStreamByIndex activates the text stream at the given index in
// the order delivered by OnStreamsReady. A negative index disables text
// rendering entirely (text flag off; no select-streams sent).
//
// Safe to call before the StreamCollection arrives: the request is
// remembered and re-applied from processCollection once streams are
// visible. Without this, a saved non-default subtitle (e.g. user picked
// stream 1) would be reflected in the dropdown but not on screen, since
// playbin3 would fall back to whatever the container marks as default.
func (c *core) SelectTextStreamByIndex(idx int) {
	if idx < 0 {
		c.SetTextEnabled(false)
		c.mu.Lock()
		c.pendingTextActive = false
		c.mu.Unlock()
		return
	}
	c.SetTextEnabled(true)
	c.mu.Lock()
	c.pendingTextIdx = idx
	c.pendingTextActive = true
	streamsReady := len(c.textStreams) > 0
	c.mu.Unlock()
	if streamsReady {
		c.selectStreamsByIndex(-1, idx)
	}
}

// SelectAudioStreamByIndex activates the audio stream at the given index.
// A negative index leaves the current audio selection alone.
//
// Safe to call before the StreamCollection arrives: the request is remembered
// and re-applied from processCollection once streams are visible.
func (c *core) SelectAudioStreamByIndex(idx int) {
	if idx < 0 {
		c.mu.Lock()
		c.pendingAudioActive = false
		c.mu.Unlock()
		return
	}
	c.mu.Lock()
	c.pendingAudioIdx = idx
	c.pendingAudioActive = true
	streamsReady := len(c.audioStreams) > 0
	c.mu.Unlock()
	if streamsReady {
		c.selectStreamsByIndex(idx, -1)
	}
}

// selectStreamsByIndex sends a SELECT_STREAMS event to playbin3 picking the
// audio/text streams at the given indices. Negative arguments mean "leave
// that type unchanged" (we'll pick whichever the previous OnStreamsReady
// reported as selected).
//
// Stream selection in playbin3 is fundamentally event-based; the legacy
// `current-text` integer property does not exist. We pass the *gst.Stream
// pointers obtained from the StreamCollection.
func (c *core) selectStreamsByIndex(audioIdx, textIdx int) {
	if c.closed.Load() {
		return
	}

	c.mu.Lock()
	pickByIdx := func(streams []streamInfo, idx int) *gst.Stream {
		if idx >= 0 {
			if idx < len(streams) {
				return streams[idx].stream
			}
			return nil
		}
		for _, s := range streams {
			if s.Selected {
				return s.stream
			}
		}
		if len(streams) > 0 {
			return streams[0].stream
		}
		return nil
	}
	pickedAudio := pickByIdx(c.audioStreams, audioIdx)
	pickedText := pickByIdx(c.textStreams, textIdx)
	pickedVideo := pickByIdx(c.videoStreams, -1)
	c.mu.Unlock()

	var streams []*gst.Stream
	if pickedVideo != nil {
		streams = append(streams, pickedVideo)
	}
	if pickedAudio != nil {
		streams = append(streams, pickedAudio)
	}
	if c.textEnabled.Load() && pickedText != nil {
		streams = append(streams, pickedText)
	}
	if len(streams) == 0 {
		return
	}
	event := gst.NewSelectStreamsEvent(streams)
	if !c.pipeline.SendEvent(event) {
		slog.Debug("player core: select-streams event was not handled")
	}
}

// Close tears down the pipeline. Safe to call multiple times.
//
// Must be called on the GTK main thread. The order is critical:
//  1. Stop the bus watch first so no late callbacks fire on a half-closed core.
//  2. Take the pipeline to NULL synchronously — this releases the sink's
//     internal frame buffers and any GL resources tied to GTK.
//  3. Only THEN unref the paintable wrapper. Unref'ing earlier would let the
//     pipeline keep using a stale paintable; unref'ing from a Go finalizer
//     would happen on a non-main goroutine and corrupt GTK state.
//
// Callers must clear the paintable from any GtkPicture displaying it
// (picture.SetPaintable((*gdk.PaintableBase)(nil))) BEFORE calling Close,
// otherwise the picture's next snapshot may reach into freed sink memory.
func (c *core) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	if c.bus != nil {
		c.bus.RemoveWatch()
		c.bus = nil
	}
	if c.pipeline != nil {
		_ = c.pipeline.SetState(gst.StateNull)
	}
	if c.paintableObj != nil {
		c.paintableObj.Unref()
		c.paintableObj = nil
	}
	c.paintable = nil
	c.videoSink = nil
	c.paintableSink = nil
	c.pipeline = nil
}

// onBusMessage is invoked on the GLib main loop. Returning true keeps the
// watch attached.
func (c *core) onBusMessage(msg *gst.Message) bool {
	if c.closed.Load() {
		return true
	}
	switch msg.Type() {
	case gst.MessageError:
		gerr := msg.ParseError()
		// gerr.DebugString() typically embeds the source URI on HTTP failures,
		// which carries the X-Plex-Token query param. Strip it before logging
		// or surfacing to OnError.
		err := fmt.Errorf("%s (%s)", redactToken(gerr.Message()), redactToken(gerr.DebugString()))
		slog.Error("player core: pipeline error", "error", err)
		if c.opts.OnError != nil {
			c.opts.OnError(err)
		}
	case gst.MessageEOS:
		slog.Debug("player core: end-of-stream")
		if c.opts.OnEOS != nil {
			c.opts.OnEOS()
		}
	case gst.MessageBuffering:
		percent := msg.ParseBuffering()
		// Pause the pipeline while the buffer fills, but only resume on
		// 100% if the user actually wants to be playing. Otherwise a
		// 100% buffering message arriving after the user hit pause would
		// silently restart playback while the UI still shows paused.
		if percent < 100 {
			_ = c.pipeline.SetState(gst.StatePaused)
		} else if c.desiredPlaying.Load() {
			_ = c.pipeline.SetState(gst.StatePlaying)
		}
		if c.opts.OnBuffering != nil {
			c.opts.OnBuffering(percent)
		}
	case gst.MessageStateChanged:
		// State-changed fires for every element in the pipeline. We only
		// care about top-level transitions; filter by element name (set by
		// the playbin3 factory, e.g. "playbin30"). Other elements are noisy
		// during preroll.
		if msg.Source() != c.pipelineName {
			return true
		}
		_, newState := msg.ParseStateChanged()
		if c.opts.OnStateChange != nil {
			c.opts.OnStateChange(newState)
		}
	case gst.MessageAsyncDone:
		c.asyncDone.Store(true)
		if c.opts.OnAsyncDone != nil {
			c.opts.OnAsyncDone()
		}
	case gst.MessageStreamCollection:
		// Available streams have just been (re)discovered — replace our
		// cached lists, re-apply any pending text-stream selection, and
		// notify the caller. The Selected flags here reflect the source's
		// *default* selection, not anything we asked for.
		collection := msg.ParseStreamCollection()
		if collection == nil {
			return true
		}
		c.processStreamCollection(collection)
	case gst.MessageStreamsSelected:
		// Active selection just changed (typically a confirmation of our
		// own SELECT_STREAMS event). Refresh Selected flags on existing
		// entries; do NOT replace the lists or re-apply pending — that
		// would risk a feedback loop if the source defaults disagree with
		// our pick.
		c.refreshSelectedFlags(msg)
	}
	return true
}

// processStreamCollection replaces the cached stream lists and applies any
// deferred stream selection.
func (c *core) processStreamCollection(collection *gst.StreamCollection) {
	defer func() {
		// Catches Go-level panics (e.g. accessing a nil receiver on a
		// half-initialised *gst.Stream wrapper) so the bus watch survives
		// to process the next message. NOTE: this does NOT catch SIGSEGV
		// raised inside cgo — those are unrecoverable and would still
		// terminate the process.
		if r := recover(); r != nil {
			slog.Warn("player core: processStreamCollection panic", "panic", r)
		}
	}()
	audio, text, video := splitStreams(collection)

	c.mu.Lock()
	c.audioStreams = audio
	c.textStreams = text
	c.videoStreams = video
	pendingAudioIdx := -1
	hasPendingAudio := c.pendingAudioActive
	if hasPendingAudio {
		pendingAudioIdx = c.pendingAudioIdx
	}
	pendingTextIdx := -1
	hasPendingText := c.pendingTextActive
	if hasPendingText {
		pendingTextIdx = c.pendingTextIdx
	}
	c.mu.Unlock()

	// Apply pending selections now that streams are visible. Skip if the
	// desired streams are already marked Selected — re-firing the event
	// would just round-trip via MessageStreamsSelected. We keep pending
	// state so future MessageStreamCollection messages (e.g. after a URI
	// swap) re-apply the user's picks rather than reverting to defaults.
	//
	// An out-of-range pending index means our caller's stream count
	// disagrees with what playbin3 just enumerated — typically because
	// Plex's metadata order and the container's stream order diverged.
	// Log it so this is visible if it ever happens; we fall back to the
	// source-default selection rather than picking a wrong stream.
	audioIdx := -1
	textIdx := -1
	needsSelect := false
	if hasPendingAudio && pendingAudioIdx >= 0 {
		if pendingAudioIdx < len(audio) {
			audioIdx = pendingAudioIdx
			needsSelect = needsSelect || !audio[pendingAudioIdx].Selected
		} else {
			slog.Warn("player core: pending audio index out of range",
				"requested", pendingAudioIdx, "available", len(audio))
		}
	}
	if hasPendingText && pendingTextIdx >= 0 {
		if pendingTextIdx < len(text) {
			textIdx = pendingTextIdx
			needsSelect = needsSelect || !text[pendingTextIdx].Selected
		} else {
			slog.Warn("player core: pending text index out of range",
				"requested", pendingTextIdx, "available", len(text))
		}
	}
	if needsSelect {
		c.selectStreamsByIndex(audioIdx, textIdx)
	}
}

// refreshSelectedFlags updates the Selected flag on cached stream entries
// from a streams-selected message, matched by StreamID. The message's
// StreamsSelected list is the authoritative active set; the StreamFlags on
// the collection describe default selection and can disagree after an
// explicit select-streams event.
func (c *core) refreshSelectedFlags(msg *gst.Message) {
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("player core: refreshSelectedFlags panic", "panic", r)
		}
	}()
	selected := make(map[string]bool)
	size := msg.StreamsSelectedSize()
	for i := uint(0); i < size; i++ {
		stream := msg.StreamsSelectedGetStream(i)
		if stream == nil {
			continue
		}
		selected[stream.StreamID()] = true
	}

	c.mu.Lock()
	for _, list := range [][]streamInfo{c.audioStreams, c.textStreams, c.videoStreams} {
		for i := range list {
			list[i].Selected = selected[list[i].StreamID]
		}
	}
	c.mu.Unlock()
}

// splitStreams partitions a StreamCollection into per-type slices in
// declaration order.
func splitStreams(collection *gst.StreamCollection) (audio, text, video []streamInfo) {
	size := collection.GetSize()
	for i := uint(0); i < size; i++ {
		stream := collection.GetStreamAt(i)
		if stream == nil {
			continue
		}
		info := streamInfo{
			StreamID: stream.StreamID(),
			Selected: stream.StreamFlags()&gst.StreamFlagSelect != 0,
			stream:   stream,
		}
		switch stream.StreamType() {
		case gst.StreamTypeAudio:
			audio = append(audio, info)
		case gst.StreamTypeText:
			text = append(text, info)
		case gst.StreamTypeVideo:
			video = append(video, info)
		}
	}
	return audio, text, video
}
