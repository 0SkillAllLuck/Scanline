package cards

import (
	"strconv"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/jwijenbergh/puregotk/v4/glib"
)

// NewMoviePreviewCard creates a new 16:9 preview card for a movie.
func NewMoviePreviewCard(metadata *sources.Metadata, artUrl, serverID string) schwifty.Button {
	var progress float64
	if metadata.Duration > 0 && metadata.ViewOffset > 0 {
		progress = float64(metadata.ViewOffset) / float64(metadata.Duration)
	}

	return previewCard(
		metadata.Title,
		subTitle(strconv.Itoa(metadata.Year)),
		artUrl,
		progress,
	).
		ActionName("win.route.movie").
		ActionTargetValue(glib.NewVariantString(serverID + "/" + metadata.RatingKey))
}
