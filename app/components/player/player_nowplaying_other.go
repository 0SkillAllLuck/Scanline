//go:build !darwin

package player

import (
	"context"

	"github.com/0skillallluck/scanline/app/sources"
)

// fetchAndPushArtwork is a no-op on non-Darwin: nowplaying.SetTextMetadata /
// SetArtwork are stubs there, so we'd just be issuing wasted HTTP fetches.
func fetchAndPushArtwork(ctx context.Context, src sources.Source, params PlayerParams) {}
