package cards

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/jwijenbergh/puregotk/v4/glib"
)

// NewShowPoster creates a new poster card for a tv-show.
func NewShowPoster(metadata *sources.Metadata, seasonCount int, coverURL, serverID string) schwifty.Button {
	return poster(
		metadata.Title,
		subTitle(gettext.GetN("%d Season", "%d Seasons", seasonCount, seasonCount)),
		coverURL,
	).
		ActionName("win.route.show").
		ActionTargetValue(glib.NewVariantString(serverID + "/" + metadata.RatingKey))
}
