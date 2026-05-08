//go:build !darwin

package nowplaying

// Configure is a no-op on non-Darwin platforms.
func Configure(info Info, handlers Handlers) {}

// SetTextMetadata is a no-op on non-Darwin platforms.
func SetTextMetadata(info Info) {}

// SetState is a no-op on non-Darwin platforms.
func SetState(state State) {}

// SetPosition is a no-op on non-Darwin platforms.
func SetPosition(positionUs int64) {}

// SetArtwork is a no-op on non-Darwin platforms.
func SetArtwork(data []byte) {}

// Clear is a no-op on non-Darwin platforms.
func Clear() {}
