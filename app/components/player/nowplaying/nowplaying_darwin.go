//go:build darwin

package nowplaying

/*
#cgo darwin CFLAGS: -fobjc-arc -fmodules
#cgo darwin LDFLAGS: -framework MediaPlayer -framework Foundation -framework AppKit

#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>

void scanline_np_init(void);
void scanline_np_set_metadata(const char *title, const char *artist,
    const char *album, double durationSec, int kind);
void scanline_np_set_state(int state);
void scanline_np_set_position(double positionSec);
void scanline_np_set_artwork(const void *data, int len);
void scanline_np_set_handler_enabled(int handlerID, bool enabled);
void scanline_np_clear(void);
*/
import "C"

import (
	"sync"
	"unsafe"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
)

// Command IDs — must stay in sync with CMD_* defines in nowplaying_darwin.m.
const (
	cmdPlayPause = 0
	cmdPlay      = 1
	cmdPause     = 2
	cmdNext      = 3
	cmdPrev      = 4
	cmdSkipFwd   = 5
	cmdSkipBack  = 6
	cmdSeekTo    = 7
	cmdStop      = 8
)

type session struct {
	h Handlers
}

var (
	mu       sync.Mutex
	current  *session
	initOnce sync.Once
)

//export scanlineNpDispatch
func scanlineNpDispatch(handlerID C.int, doubleArg C.double) {
	id := int(handlerID)
	arg := float64(doubleArg)
	// MediaPlayer.framework calls us on the AppKit main runloop, but the
	// player handlers call into GStreamer / GTK so we hop onto the GLib main
	// loop. Re-check current under the lock inside the hop because the
	// session can rotate between the OS callback and the idle-tick.
	schwifty.OnMainThreadOncePure(func() {
		mu.Lock()
		s := current
		mu.Unlock()
		if s == nil {
			return
		}
		switch id {
		case cmdPlayPause:
			if s.h.PlayPause != nil {
				s.h.PlayPause()
			}
		case cmdPlay:
			if s.h.Play != nil {
				s.h.Play()
			}
		case cmdPause:
			if s.h.Pause != nil {
				s.h.Pause()
			}
		case cmdNext:
			if s.h.Next != nil {
				s.h.Next()
			}
		case cmdPrev:
			if s.h.Previous != nil {
				s.h.Previous()
			}
		case cmdSkipFwd:
			if s.h.SkipFwd != nil {
				s.h.SkipFwd(arg)
			}
		case cmdSkipBack:
			if s.h.SkipBack != nil {
				s.h.SkipBack(arg)
			}
		case cmdSeekTo:
			if s.h.SeekTo != nil {
				s.h.SeekTo(int64(arg * 1e6))
			}
		case cmdStop:
			if s.h.Stop != nil {
				s.h.Stop()
			}
		}
	})
}

// Configure registers the active player session. Idempotent — later calls
// replace the current session. The OS-side target blocks are installed once
// and dispatch through the package-global current pointer.
func Configure(info Info, h Handlers) {
	initOnce.Do(func() {
		C.scanline_np_init()
	})
	mu.Lock()
	current = &session{h: h}
	mu.Unlock()
	pushMetadata(info)
	C.scanline_np_set_handler_enabled(C.int(cmdPlayPause), C.bool(h.PlayPause != nil))
	C.scanline_np_set_handler_enabled(C.int(cmdPlay), C.bool(h.Play != nil))
	C.scanline_np_set_handler_enabled(C.int(cmdPause), C.bool(h.Pause != nil))
	C.scanline_np_set_handler_enabled(C.int(cmdNext), C.bool(h.Next != nil))
	C.scanline_np_set_handler_enabled(C.int(cmdPrev), C.bool(h.Previous != nil))
	C.scanline_np_set_handler_enabled(C.int(cmdSkipFwd), C.bool(h.SkipFwd != nil))
	C.scanline_np_set_handler_enabled(C.int(cmdSkipBack), C.bool(h.SkipBack != nil))
	C.scanline_np_set_handler_enabled(C.int(cmdSeekTo), C.bool(h.SeekTo != nil))
	C.scanline_np_set_handler_enabled(C.int(cmdStop), C.bool(h.Stop != nil))
}

// SetTextMetadata refreshes the title / artist / album / kind / duration
// fields without touching state, position, or artwork. Used to apply
// late-arriving show + season titles after the async metadata fetch lands.
func SetTextMetadata(info Info) {
	pushMetadata(info)
}

func pushMetadata(info Info) {
	cTitle := C.CString(info.Title)
	defer C.free(unsafe.Pointer(cTitle))
	cArtist := C.CString(info.Artist)
	defer C.free(unsafe.Pointer(cArtist))
	cAlbum := C.CString(info.AlbumTitle)
	defer C.free(unsafe.Pointer(cAlbum))
	C.scanline_np_set_metadata(cTitle, cArtist, cAlbum,
		C.double(info.DurationUs)/1e6, C.int(info.Kind))
}

// SetState publishes the current playback state. Called from the player's
// OnStateChange callback.
func SetState(state State) {
	C.scanline_np_set_state(C.int(state))
}

// SetPosition publishes the elapsed playback time. Called from the existing
// 500ms progress ticker. Does not touch the playback rate — that's owned by
// SetState — so a paused tick won't accidentally set rate=1.0.
func SetPosition(positionUs int64) {
	C.scanline_np_set_position(C.double(positionUs) / 1e6)
}

// SetArtwork attaches cover-art bytes (PNG or JPEG). Pass nil to leave the
// existing artwork in place. Decoded into MPMediaItemArtwork via NSImage on
// the ObjC side.
func SetArtwork(data []byte) {
	if len(data) == 0 {
		return
	}
	C.scanline_np_set_artwork(unsafe.Pointer(&data[0]), C.int(len(data)))
}

// Clear tears down the published Now Playing info and disables every remote
// command. Safe to call multiple times. Called from the player's cleanup
// path so the Control Center widget vanishes the moment the player closes.
func Clear() {
	mu.Lock()
	current = nil
	mu.Unlock()
	C.scanline_np_clear()
}
