package cards

import (
	"strconv"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/jwijenbergh/puregotk/v4/glib"
)

// NewMoviePoster creates a new poster card for a movie.
func NewMoviePoster(metadata *sources.Metadata, coverURL, serverID string) schwifty.Button {
	return poster(
		metadata.Title,
		subTitle(strconv.Itoa(metadata.Year)),
		coverURL,
	).
		ActionName("win.route.movie").
		ActionTargetValue(glib.NewVariantString(serverID + "/" + metadata.RatingKey))
}
