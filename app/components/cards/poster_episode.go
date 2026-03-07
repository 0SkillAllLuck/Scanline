package cards

import (
	"fmt"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"github.com/0skillallluck/scanline/app/sources"
)

// NewEpisodePoster creates a new poster card for a tv-show episode.
func NewEpisodePoster(metadata *sources.Metadata, coverURL, serverID string) schwifty.Button {
	return poster(
		metadata.GrandparentTitle,
		VStack(
			subTitle(metadata.Title),
			subTitle(fmt.Sprintf("S%d - E%d", metadata.ParentIndex, metadata.Index)),
		),
		coverURL,
	).
		ActionName("win.route.episode").
		ActionTargetValue(glib.NewVariantString(serverID + "/" + metadata.RatingKey))
}
