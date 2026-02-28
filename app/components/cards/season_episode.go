package cards

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/internal/resources"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/utils/imageutils"
	"github.com/jwijenbergh/puregotk/v4/glib"
	"github.com/jwijenbergh/puregotk/v4/gtk"
	"github.com/jwijenbergh/puregotk/v4/pango"
)

func NewSeasonEpisode(metadata *sources.Metadata, coverUrl, serverID string) schwifty.Button {
	return Button().
		Child(
			VStack(
				Picture().
					SizeRequest(320, 180).
					ContentFit(gtk.ContentFitCoverValue).
					FromPaintable(resources.MissingAlbum()).
					ConnectRealize(func(w gtk.Widget) {
						if preference.Performance().AllowPreviewImages() {
							imageutils.LoadIntoPictureScaled(coverUrl, 320, 180, gtk.PictureNewFromInternalPtr(w.Ptr))
						}
					}).
					CSS("picture { min-width: 320px; min-height: 180px; }").
					CornerRadius(10).Overflow(gtk.OverflowHiddenValue),
				Label(metadata.Title).
					WithCSSClass("heading").
					MarginTop(10).
					MaxWidthChars(30).
					HAlign(gtk.AlignStartValue).
					Ellipsis(pango.EllipsizeEndValue),
				subTitle(gettext.Get("Episode %d", metadata.Index)).
					MarginTop(2),
			),
		).
		Padding(10).
		HExpand(false).
		VExpand(false).
		WithCSSClass("flat").
		ActionName("win.route.episode").
		ActionTargetValue(glib.NewVariantString(serverID + "/" + metadata.RatingKey))
}
