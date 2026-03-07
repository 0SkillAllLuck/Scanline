package cards

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
)

// NewSeasonPoster creates a new poster card for a tv-show season.
func NewSeasonPoster(metadata *sources.Metadata, coverURL, serverID string) schwifty.Button {
	var progress float64
	if metadata.LeafCount > 0 {
		progress = float64(metadata.ViewedLeafCount) / float64(metadata.LeafCount)
	}

	return posterWithProgress(
		metadata.Title,
		subTitle(gettext.GetN("%d Episode", "%d Episodes", metadata.LeafCount, metadata.LeafCount)),
		coverURL,
		progress,
	).
		ActionName("win.route.season").
		ActionTargetValue(glib.NewVariantString(serverID + "/" + metadata.RatingKey))
}
