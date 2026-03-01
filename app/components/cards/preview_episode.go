package cards

import (
	"fmt"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/jwijenbergh/puregotk/v4/glib"
)

// NewEpisodePreviewCard creates a new 16:9 preview card for a tv-show episode.
func NewEpisodePreviewCard(metadata *sources.Metadata, artURL, serverID string) schwifty.Button {
	var progress float64
	if metadata.Duration > 0 && metadata.ViewOffset > 0 {
		progress = float64(metadata.ViewOffset) / float64(metadata.Duration)
	}

	return previewCard(
		metadata.GrandparentTitle,
		VStack(
			subTitle(metadata.Title),
			subTitle(fmt.Sprintf("S%d · E%d", metadata.ParentIndex, metadata.Index)),
		),
		artURL,
		progress,
	).
		ActionName("win.route.episode").
		ActionTargetValue(glib.NewVariantString(serverID + "/" + metadata.RatingKey))
}
