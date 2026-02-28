package cards

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/jwijenbergh/puregotk/v4/glib"
)

// NewSeasonPoster creates a new poster card for a tv-show season.
func NewSeasonPoster(metadata *sources.Metadata, coverUrl, serverID string) schwifty.Button {
	return poster(
		metadata.Title,
		subTitle(gettext.GetN("%d Episode", "%d Episodes", metadata.LeafCount, metadata.LeafCount)),
		coverUrl,
	).
		ActionName("win.route.season").
		ActionTargetValue(glib.NewVariantString(serverID + "/" + metadata.RatingKey))
}
