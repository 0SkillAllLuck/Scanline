package cards

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/gdk"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"codeberg.org/puregotk/puregotk/v4/pango"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/utils/imageutils"
)

func NewSeasonEpisode(metadata *sources.Metadata, coverURL, serverID string) schwifty.Button {
	var progress float64
	if metadata.ViewCount > 0 && metadata.ViewOffset == 0 {
		progress = 1.0
	} else if metadata.Duration > 0 && metadata.ViewOffset > 0 {
		progress = float64(metadata.ViewOffset) / float64(metadata.Duration)
	}

	progressWidth := int32(320 * progress)

	picture := Picture().
		SizeRequest(320, 180).
		ContentFit(gtk.ContentFitCoverValue).
		FromPaintable(gdk.NewTextureFromResource("/dev/skillless/Scanline/icons/scalable/state/missing-album.svg")).
		ConnectRealize(func(w gtk.Widget) {
			if preference.Performance().AllowPreviewImages() {
				imageutils.LoadIntoPictureScaled(coverURL, 320, 180, gtk.PictureNewFromInternalPtr(w.Ptr))
			}
		})

	progressBar := Box(gtk.OrientationHorizontalValue).
		SizeRequest(progressWidth, 4).
		VAlign(gtk.AlignEndValue).
		HAlign(gtk.AlignStartValue).
		CSS("box { background-color: @accent_bg_color; }")

	imageContainer := Bin().
		Child(Overlay(picture).AddOverlay(progressBar)).
		SizeRequest(320, 180).
		CornerRadius(10).
		Overflow(gtk.OverflowHiddenValue)

	return Button().
		Child(
			VStack(
				imageContainer,
				Label(metadata.Title).
					WithCSSClass("heading").
					MarginTop(10).
					MaxWidthChars(30).
					HAlign(gtk.AlignStartValue).
					Ellipsis(pango.EllipsizeEndValue),
				subTitle(gettext.Getf("Episode %d", metadata.Index)).
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
