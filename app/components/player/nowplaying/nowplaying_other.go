//go:build !darwin

// Stubs for non-Darwin builds; every exported function is a no-op.
package nowplaying

func Configure(info Info, handlers Handlers) {}
func SetTextMetadata(info Info)              {}
func SetState(state State)                   {}
func SetPosition(positionUs int64)           {}
func SetArtwork(data []byte)                 {}
func Clear()                                 {}
