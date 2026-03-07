package cards

import (
	"strconv"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"github.com/0skillallluck/scanline/app/sources"
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
