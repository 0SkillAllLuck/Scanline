// Package nowplaying integrates Scanline with the macOS Control Center
// "Now Playing" widget and MPRemoteCommandCenter for hardware media keys.
//
// On non-Darwin builds every exported function is a no-op (see the build-tag
// shadowed file). On Darwin, the implementation lives in
// nowplaying_darwin.{go,m} and bridges to MediaPlayer.framework via cgo.
package nowplaying

// State enumerates the playback states we report to the OS.
type State int

const (
	StatePlaying State = iota
	StatePaused
	StateStopped
)

// MediaKind distinguishes movies from episodes for the OS media-type field.
type MediaKind int

const (
	KindMovie MediaKind = iota
	KindEpisode
)

// Info is the per-session metadata we publish to MPNowPlayingInfoCenter.
// Title is required; Artist / AlbumTitle are optional and typically only set
// for episodes (show name and season label respectively). Duration is in
// microseconds to match the player core's internal clock.
type Info struct {
	Title      string
	Artist     string
	AlbumTitle string
	Kind       MediaKind
	DurationUs int64
}

// Handlers receives commands from MPRemoteCommandCenter (media keys, the
// Control Center widget, Bluetooth remotes, AirPods clicker). All handlers
// are invoked on the GTK main thread.
//
// A nil handler disables the corresponding command on the OS side. Handlers
// taking a seconds argument receive the user-configured skip interval (we
// hint 15s by default). SeekTo receives the target position in microseconds.
type Handlers struct {
	PlayPause func()
	Play      func()
	Pause     func()
	Next      func()
	Previous  func()
	SkipFwd   func(seconds float64)
	SkipBack  func(seconds float64)
	SeekTo    func(positionUs int64)
	Stop      func()
}
